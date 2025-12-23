/** @type {import('tailwindcss').Config} */
export default {
  content: [
    './src/**/*.{html,js,svelte,ts}',
    './src/lib/**/*.{html,js,svelte,ts}', // Ensure lib components are scanned
  ],
  theme: {
    extend: {},
  },
  plugins: [],
};
