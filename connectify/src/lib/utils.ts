import { type ClassValue, clsx } from "clsx"
import { twMerge } from "tailwind-merge"
import type { SvelteHTMLElements } from "svelte/elements";
 
export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

// WithElementRef: A utility type to extend HTML element attributes with a 'ref' property.
// This is useful for components that need to forward refs to their underlying DOM elements.
// It works by taking a Svelte HTML element type and adding an optional 'ref' property
// that matches the type of the element's 'ref' attribute.
export type WithElementRef<T extends SvelteHTMLElements[keyof SvelteHTMLElements]> = T & {
  ref?: T["ref"];
};

// WithoutChildrenOrChild: Used to remove 'children' or 'child' props from a type,
// often when using Svelte's new slot syntax.
// This is a simplified version; a more robust one might be needed depending on exact usage.
export type WithoutChildrenOrChild<T> = Omit<T, "children" | "child">;

export type WithoutChildren<T> = Omit<T, 'children' | 'child'>;

export type WithoutChild<T> = Omit<T, 'children' | 'child'>;


;
