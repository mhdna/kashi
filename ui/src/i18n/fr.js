// French for the app: English text -> French. Each file in ./fr holds one area
// (default export: { 'English': 'Français' }); {placeholders} must stay as they are.
const files = import.meta.glob('./fr/*.js', { eager: true })

export const FR = Object.assign({}, ...Object.values(files).map(m => m.default))
