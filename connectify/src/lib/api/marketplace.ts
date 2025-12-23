import { apiRequest } from '$lib/api';
import type { UserShortResponse } from '$lib/api'; // Reuse generic types

export type ProductStatus = 'available' | 'sold' | 'pending' | 'archived';

export interface Category {
    id: string;
    name: string;
    slug: string;
    icon: string;
    order: number;
}

export interface Product {
    id: string;
    seller_id: string;
    category_id: string;
    title: string;
    description: string;
    price: number;
    currency: string;
    images: string[];
    location: string;
    status: ProductStatus;
    tags?: string[];
    views: number;
    created_at: string;
    updated_at: string;
    seller?: UserShortResponse; // Populated seller info
    category?: Category;    // Populated category info
    is_saved?: boolean;
}

export interface CreateProductRequest {
    title: string;
    description: string;
    price: number;
    currency: string;
    category_id: string;
    images: string[];
    location: string;
    tags?: string[];
}

export interface ProductFilter {
    q?: string;
    category_id?: string;
    min_price?: number;
    max_price?: number;
    location?: string;
    sort_by?: 'price_asc' | 'price_desc' | 'newest';
    page?: number;
    limit?: number;
}

export interface MarketplaceListResponse {
    products: Product[];
    total: number;
    page: number;
    limit: number;
}

export async function getCategories(): Promise<Category[]> {
    return apiRequest('GET', '/marketplace/categories', undefined, true);
}

export async function createProduct(data: CreateProductRequest): Promise<Product> {
    return apiRequest('POST', '/marketplace/products', data, true);
}

export async function getProducts(filter: ProductFilter = {}): Promise<MarketplaceListResponse> {
    const params = new URLSearchParams();
    if (filter.q) params.set('q', filter.q);
    if (filter.category_id) params.set('category_id', filter.category_id);
    if (filter.min_price) params.set('min_price', String(filter.min_price));
    if (filter.max_price) params.set('max_price', String(filter.max_price));
    if (filter.location) params.set('location', filter.location);
    if (filter.sort_by) params.set('sort_by', filter.sort_by);
    if (filter.page) params.set('page', String(filter.page));
    if (filter.limit) params.set('limit', String(filter.limit));

    return apiRequest('GET', `/marketplace/products?${params.toString()}`, undefined, true);
}

export async function getProduct(id: string): Promise<Product> {
    return apiRequest('GET', `/marketplace/products/${id}`, undefined, true);
}

export async function deleteProduct(id: string): Promise<void> {
    return apiRequest('DELETE', `/marketplace/products/${id}`, undefined, true);
}

export async function markProductSold(id: string): Promise<void> {
    return apiRequest('POST', `/marketplace/products/${id}/sold`, undefined, true);
}

export async function toggleSaveProduct(id: string): Promise<{ saved: boolean }> {
    return apiRequest('POST', `/marketplace/products/${id}/save`, undefined, true);
}

export async function getMarketplaceConversations(): Promise<import('$lib/api').ConversationSummary[]> {
    return apiRequest('GET', '/marketplace/conversations', undefined, true);
}
