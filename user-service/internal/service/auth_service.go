package service

import (
	"context"
	"errors"
	"log"
	"time"
	"user-service/config"
	"user-service/internal/repository"

	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
	"github.com/MuhibNayem/connectify-v2/shared-entity/models"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	userRepo    *repository.UserRepository
	graphRepo   *repository.GraphRepository
	redisClient *redis.Client
	cfg         *config.Config
}

func NewAuthService(
	userRepo *repository.UserRepository,
	graphRepo *repository.GraphRepository,
	redisClient *redis.Client,
	cfg *config.Config,
) *AuthService {
	return &AuthService{
		userRepo:    userRepo,
		graphRepo:   graphRepo,
		redisClient: redisClient,
		cfg:         cfg,
	}
}

func (s *AuthService) Register(ctx context.Context, user *models.User) (*models.AuthResponse, error) {
	if u, _ := s.userRepo.FindUserByEmail(ctx, user.Email); u != nil {
		return nil, errors.New("email already exists")
	}
	if u, _ := s.userRepo.FindUserByUserName(ctx, user.Username); u != nil {
		return nil, errors.New("username already exists")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	user.Password = string(hashedPassword)
	user.SetDefaultPrivacySettings() // Ensure this method exists given common models

	createdUser, err := s.userRepo.CreateUser(ctx, user)
	if err != nil {
		return nil, err
	}

	s.enqueueGraphSync(createdUser.ID)

	accessToken, refreshToken, err := s.generateTokens(ctx, createdUser)
	if err != nil {
		return nil, err
	}

	return &models.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         createdUser.ToSafeResponse(),
	}, nil
}

func (s *AuthService) Login(ctx context.Context, email, password string) (*models.AuthResponse, error) {
	user, err := s.userRepo.FindUserByEmail(ctx, email)
	if err != nil {
		return nil, errors.New("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, errors.New("invalid credentials")
	}

	accessToken, refreshToken, err := s.generateTokens(ctx, user)
	if err != nil {
		return nil, err
	}

	return &models.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         user.ToSafeResponse(),
	}, nil
}

func (s *AuthService) RefreshToken(ctx context.Context, refreshToken string) (*models.AuthResponse, error) {
	token, err := jwt.Parse(refreshToken, func(token *jwt.Token) (interface{}, error) {
		return []byte(s.cfg.JWTSecret), nil
	})
	if err != nil || !token.Valid {
		return nil, errors.New("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || claims["type"] != "refresh" {
		return nil, errors.New("invalid token type")
	}

	userIDStr := claims["id"].(string)
	storedToken, err := s.redisClient.Get(ctx, "refresh:"+userIDStr).Result()
	if err != nil || storedToken != refreshToken {
		return nil, errors.New("invalid refresh token")
	}

	userID, _ := primitive.ObjectIDFromHex(userIDStr)
	user, err := s.userRepo.FindUserByID(ctx, userID)
	if err != nil {
		return nil, errors.New("user not found")
	}

	accessToken, newRefreshToken, err := s.generateTokens(ctx, user)
	if err != nil {
		return nil, err
	}

	return &models.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		User:         user.ToSafeResponse(),
	}, nil
}

func (s *AuthService) Logout(ctx context.Context, userID, accessToken string) error {
	// Blacklist access token
	if err := s.redisClient.Set(ctx, "blacklist:"+accessToken, "1", s.cfg.AccessTokenTTL).Err(); err != nil {
		return err
	}
	// Remove refresh token
	return s.redisClient.Del(ctx, "refresh:"+userID).Err()
}

func (s *AuthService) enqueueGraphSync(userID primitive.ObjectID) {
	if s.graphRepo == nil {
		return
	}
	go func() {
		backoff := []time.Duration{0, time.Second, 3 * time.Second}
		for attempt, waitDuration := range backoff {
			if waitDuration > 0 {
				time.Sleep(waitDuration)
			}
			syncCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			err := s.graphRepo.SyncUser(syncCtx, userID)
			cancel()
			if err == nil {
				return
			}
			log.Printf("Failed to sync user to graph (attempt %d): %v", attempt+1, err)
		}
	}()
}

func (s *AuthService) generateTokens(ctx context.Context, user *models.User) (string, string, error) {
	accessClaims := jwt.MapClaims{
		"id":    user.ID.Hex(),
		"email": user.Email,
		"type":  "access",
		"exp":   time.Now().Add(s.cfg.AccessTokenTTL).Unix(),
	}
	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims).SignedString([]byte(s.cfg.JWTSecret))
	if err != nil {
		return "", "", err
	}

	refreshClaims := jwt.MapClaims{
		"id":   user.ID.Hex(),
		"type": "refresh",
		"exp":  time.Now().Add(s.cfg.RefreshTokenTTL).Unix(),
	}
	refreshToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims).SignedString([]byte(s.cfg.JWTSecret))
	if err != nil {
		return "", "", err
	}

	if err := s.redisClient.Set(ctx, "refresh:"+user.ID.Hex(), refreshToken, s.cfg.RefreshTokenTTL).Err(); err != nil {
		return "", "", err
	}

	return accessToken, refreshToken, nil
}
