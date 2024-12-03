/** @type {import('tailwindcss').Config} */
export default {
  content: ['**/*.md', '**/*.templ', '**/*.go'],
  theme: {
    extend: {},
  },
  plugins: [
    require('@tailwindcss/typography'),
    require('@tailwindcss/container-queries'),
    require('daisyui'),
  ],
};