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
  daisyui: {
    themes: true, // false: only light + dark | true: all themes | array: specific themes like this ["light", "dark", "cupcake"]
    darkTheme: "dark", // name of one of the included themes for dark mode
  },
};