import { openDB, type DBSchema } from 'idb';

interface ConnectifyDB extends DBSchema {
    keys: {
        key: string;
        value: CryptoKey;
    };
}

const DB_NAME = 'connectify-e2ee';
const STORE_NAME = 'keys';

async function getDB() {
    return openDB<ConnectifyDB>(DB_NAME, 1, {
        upgrade(db) {
            db.createObjectStore(STORE_NAME);
        }
    });
}

export async function savePrivateKey(key: CryptoKey): Promise<void> {
    const db = await getDB();
    await db.put(STORE_NAME, key, 'privateKey');
}

export async function loadPrivateKey(): Promise<CryptoKey | undefined> {
    const db = await getDB();
    return await db.get(STORE_NAME, 'privateKey');
}

export async function savePublicKey(key: CryptoKey): Promise<void> {
    const db = await getDB();
    await db.put(STORE_NAME, key, 'publicKey');
}

export async function loadPublicKey(): Promise<CryptoKey | undefined> {
    const db = await getDB();
    return await db.get(STORE_NAME, 'publicKey');
}

export async function clearKeys(): Promise<void> {
    const db = await getDB();
    await db.clear(STORE_NAME);
}
