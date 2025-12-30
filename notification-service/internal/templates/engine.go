package templates

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"sync"
)

// TemplateEngine handles notification template rendering
type TemplateEngine struct {
	templates map[string]*template.Template
	mu        sync.RWMutex
}

func NewTemplateEngine() *TemplateEngine {
	return &TemplateEngine{
		templates: make(map[string]*template.Template),
	}
}

// RegisterTemplate registers a new template
func (te *TemplateEngine) RegisterTemplate(name, templateStr string) error {
	tmpl, err := template.New(name).Parse(templateStr)
	if err != nil {
		return fmt.Errorf("failed to parse template %s: %w", name, err)
	}

	te.mu.Lock()
	te.templates[name] = tmpl
	te.mu.Unlock()

	return nil
}

// Render renders a template with the given data
func (te *TemplateEngine) Render(ctx context.Context, templateName string, data map[string]interface{}) (string, error) {
	te.mu.RLock()
	tmpl, exists := te.templates[templateName]
	te.mu.RUnlock()

	if !exists {
		return "", fmt.Errorf("template %s not found", templateName)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

// LoadDefaultTemplates loads default notification templates
func (te *TemplateEngine) LoadDefaultTemplates() error {
	defaultTemplates := map[string]string{
		"welcome": `
			<h1>Welcome, {{.UserName}}!</h1>
			<p>Thank you for joining our service.</p>
		`,
		"order_shipped": `
			<h2>Your order has shipped!</h2>
			<p>Order #{{.OrderID}} is on its way.</p>
			<p>Tracking: {{.TrackingNumber}}</p>
		`,
		"payment_received": `
			<h2>Payment Received</h2>
			<p>We've received your payment of ${{.Amount}}.</p>
			<p>Transaction ID: {{.TransactionID}}</p>
		`,
		"friend_request": `
			<h2>New Friend Request</h2>
			<p>{{.FromUser}} wants to connect with you.</p>
		`,
	}

	for name, tmplStr := range defaultTemplates {
		if err := te.RegisterTemplate(name, tmplStr); err != nil {
			return err
		}
	}

	return nil
}
