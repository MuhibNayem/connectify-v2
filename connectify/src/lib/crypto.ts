import { browser } from '$app/environment';

// E2EE Configuration
const KEY_PAIR_ALGORITHM = {
    name: 'ECDH',
    namedCurve: 'P-384'
};

const ENCRYPTION_ALGORITHM = {
    name: 'AES-GCM',
    length: 256
};

// Types
export interface KeyPair {
    publicKey: CryptoKey;
    privateKey: CryptoKey;
}

export interface EncryptedMessage {
    ciphertext: string; // Base64
    iv: string; // Base64
}

export interface BackupKeys {
    encryptedPrivateKey: string; // Base64
    iv: string; // Base64
    salt: string; // Base64
}

// ---- Key Management ----

/**
 * Generates a new ECDH Key Pair.
 * The private key is non-extractable by default, but we will make it extractable
 * momentarily ONLY for the backup process, then ideally store it in IndexedDB.
 */
export async function generateKeyPair(): Promise<KeyPair> {
    return (await window.crypto.subtle.generateKey(
        KEY_PAIR_ALGORITHM,
        true, // Must be extractable to back it up
        ['deriveKey', 'deriveBits']
    )) as KeyPair;
}

/**
 * Exports a public key to a generic raw/spki format for sharing.
 */
export async function exportPublicKey(key: CryptoKey): Promise<string> {
    const exported = await window.crypto.subtle.exportKey('spki', key);
    return arrayBufferToBase64(exported);
}

/**
 * Imports a public key from the server.
 */
export async function importPublicKey(pemOrBase64: string): Promise<CryptoKey> {
    // For simplicity, assuming Base64 SPKI
    const binary = base64ToArrayBuffer(pemOrBase64);
    return await window.crypto.subtle.importKey(
        'spki',
        binary,
        KEY_PAIR_ALGORITHM,
        true,
        []
    );
}

// ---- Encryption / Decryption  (Message Level) ----

/**
 * Derives a shared symmetric key (AES-GCM) from local private key and remote public key.
 */
export async function deriveSharedSecret(
    privateKey: CryptoKey,
    publicKey: CryptoKey
): Promise<CryptoKey> {
    return await window.crypto.subtle.deriveKey(
        {
            name: 'ECDH',
            public: publicKey
        },
        privateKey,
        ENCRYPTION_ALGORITHM,
        false, // Shared secret is NOT extractable
        ['encrypt', 'decrypt']
    );
}

/**
 * Encrypts a text message using a shared symmetric key.
 */
export async function encryptMessage(
    text: string,
    sharedKey: CryptoKey
): Promise<EncryptedMessage> {
    const iv = window.crypto.getRandomValues(new Uint8Array(12)); // 96-bit IV for GCM
    const encodedText = new TextEncoder().encode(text);

    const ciphertext = await window.crypto.subtle.encrypt(
        {
            name: 'AES-GCM',
            iv: iv
        },
        sharedKey,
        encodedText
    );

    return {
        ciphertext: arrayBufferToBase64(ciphertext),
        iv: arrayBufferToBase64(iv)
    };
}

/**
 * Decrypts a message using a shared symmetric key.
 */
export async function decryptMessage(
    ciphertextBase64: string,
    ivBase64: string,
    sharedKey: CryptoKey
): Promise<string> {
    const ciphertext = base64ToArrayBuffer(ciphertextBase64);
    const iv = base64ToArrayBuffer(ivBase64);

    try {
        const decrypted = await window.crypto.subtle.decrypt(
            {
                name: 'AES-GCM',
                iv: iv
            },
            sharedKey,
            ciphertext
        );
        return new TextDecoder().decode(decrypted);
    } catch (e) {
        console.error('Decryption failed:', e);
        return '[Decryption Error]';
    }
}

// ---- Key Backup (Password Based) ----

/**
 * Encrypts the Private Key using a password (PBKDF2 derivation).
 * This creates a safe backup that can be stored on the server.
 */
export async function encryptPrivateKeyWithPassword(
    privateKey: CryptoKey,
    password: string
): Promise<BackupKeys> {
    // 1. Export Private Key to PKCS#8
    const keyData = await window.crypto.subtle.exportKey('pkcs8', privateKey);

    // 2. Generate Salt
    const salt = window.crypto.getRandomValues(new Uint8Array(16));

    // 3. Derive Key from Password
    const passwordKey = await derivePasswordKey(password, salt);

    // 4. Encrypt
    const iv = window.crypto.getRandomValues(new Uint8Array(12));
    const ciphertext = await window.crypto.subtle.encrypt(
        {
            name: 'AES-GCM',
            iv: iv
        },
        passwordKey,
        keyData
    );

    return {
        encryptedPrivateKey: arrayBufferToBase64(ciphertext),
        iv: arrayBufferToBase64(iv),
        salt: arrayBufferToBase64(salt)
    };
}

/**
 * Decrypts the Private Key using a password.
 */
export async function decryptPrivateKeyWithPassword(
    encryptedPrivateKeyBase64: string,
    ivBase64: string,
    saltBase64: string,
    password: string
): Promise<CryptoKey> {
    const ciphertext = base64ToArrayBuffer(encryptedPrivateKeyBase64);
    const iv = base64ToArrayBuffer(ivBase64);
    const salt = base64ToArrayBuffer(saltBase64);

    // 1. Derive Key from Password
    const passwordKey = await derivePasswordKey(password, salt);

    // 2. Decrypt
    const keyData = await window.crypto.subtle.decrypt(
        {
            name: 'AES-GCM',
            iv: iv
        },
        passwordKey,
        ciphertext
    );

    // 3. Import Private Key
    return await window.crypto.subtle.importKey(
        'pkcs8',
        keyData,
        KEY_PAIR_ALGORITHM,
        true, // Must be extractable to re-export if needed
        ['deriveKey', 'deriveBits']
    );
}

// Helper: PBKDF2 Key Derivation
async function derivePasswordKey(password: string, salt: Uint8Array): Promise<CryptoKey> {
    const encoder = new TextEncoder();
    const keyMaterial = await window.crypto.subtle.importKey(
        'raw',
        encoder.encode(password),
        'PBKDF2',
        false,
        ['deriveKey']
    );

    return await window.crypto.subtle.deriveKey(
        {
            name: 'PBKDF2',
            salt: salt,
            iterations: 100000,
            hash: 'SHA-256'
        },
        keyMaterial,
        { name: 'AES-GCM', length: 256 },
        false,
        ['encrypt', 'decrypt']
    );
}

// ---- Utilities ----

function arrayBufferToBase64(buffer: ArrayBuffer): string {
    let binary = '';
    const bytes = new Uint8Array(buffer);
    const len = bytes.byteLength;
    for (let i = 0; i < len; i++) {
        binary += String.fromCharCode(bytes[i]);
    }
    return window.btoa(binary);
}

function base64ToArrayBuffer(base64: string): ArrayBuffer {
    const binary_string = window.atob(base64);
    const len = binary_string.length;
    const bytes = new Uint8Array(len);
    for (let i = 0; i < len; i++) {
        bytes[i] = binary_string.charCodeAt(i);
    }
    return bytes.buffer;
}

/**
 * Computes a visual fingerprint (Safety Number) for a public key.
 * Uses SHA-256 of the SPKI key data.
 * Returns a hex string formatted in groups for readability.
 */
export async function computeFingerprint(publicKeyBase64: string): Promise<string> {
    const rawData = base64ToArrayBuffer(publicKeyBase64);
    const hash = await window.crypto.subtle.digest('SHA-256', rawData);

    // Convert to Hex
    const hashArray = Array.from(new Uint8Array(hash));
    const hashHex = hashArray.map(b => b.toString(16).padStart(2, '0')).join('').toUpperCase();

    // Format: 4 blocks of 8 characters (first 32 chars of hash is usually enough for visual check, 
    // but full 64 chars is safer. Let's show first 32 chars in 4 blocks of 8)
    // Signal uses numeric 60-digit codes. Hex is fine for us.
    // Let's use the full hash but chunked.
    // 64 chars. 8 chunks of 8.
    const chunks = hashHex.match(/.{1,8}/g);
    return chunks ? chunks.join(' ') : hashHex;
}
