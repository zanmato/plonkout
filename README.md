# Plonkout

A Progressive Web App (PWA) for workout tracking built with Vue 3 and modern web technologies.

## Features

- 🏋️ **Comprehensive Exercise Database**: Pre-loaded with armwrestling, bodybuilding, powerlifting, and cardio exercises
- 💪 **Muscle Group Organization**: Exercises grouped by muscle groups (Forearm, Chest, Back, Shoulders, Legs, Biceps, Triceps, Abs)
- ⚡ **Exercise Types**: Support for both strength training and cardio exercises
- 📱 **Mobile-First Design**: iOS-style interface optimized for mobile devices
- 🎨 **Neobrutalism UI**: Bold, modern design with Tailwind CSS
- 🌐 **Progressive Web App**: Installable, works offline, with service worker caching
- 🌍 **Internationalization**: Support for multiple languages (English, Swedish)
- ⚙️ **Flexible Settings**: Customizable weight units, exercise display modes, themes
- 📊 **Statistics**: Visual charts and analytics for workout tracking
- 💾 **Local Storage**: All data stored locally using IndexedDB
- 🔄 **Auto-save**: Automatic saving with debounced updates

## Development

### Prerequisites

- Node.js 22+
- pnpm

### Setup

```bash
# Install dependencies
pnpm install

# Start development server
pnpm dev

# Run tests
pnpm test

# Build for production
pnpm build

# Run linter
pnpm lint
```

## Project Structure

```
src/
├── components/     # Reusable Vue components
├── views/          # Main application views
├── utils/          # Utility functions and database operations
├── assets/         # Static assets and global styles
├── locales/        # i18n translation files
├── volt/           # Custom UI component library
├── composables/    # Vue composables
└── tests/          # Test files
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for new functionality
5. Ensure all tests pass and linting is clean
6. Submit a pull request

## License

This project is open source and available under the [MIT License](LICENSE).
