# BMJ Polling Frontend

A minimal, mobile-first React application for creating and participating in Balanced Majority Judgment (BMJ) polls.

## Setup

This project was initialized with Vite + React + TypeScript and configured for optimal performance.

### Prerequisites

- Node.js 18+ 
- npm

### Installation

```bash
npm install
```

### Development

```bash
npm run dev
```

The development server will start at http://localhost:5173

### Build

```bash
npm run build
```

Builds the app for production to the `dist` folder.

### Environment Variables

Copy `.env.example` to `.env` and configure:

- `VITE_API_BASE_URL`: Base URL for the API server (default: http://localhost:8080)

## Project Structure

```
src/
├── components/
│   ├── common/          # Reusable UI components
│   ├── poll/            # Poll-specific components  
│   └── layout/          # Layout components
├── pages/               # Route components
├── api/                 # API client and functions
├── hooks/               # Custom React hooks
├── types/               # TypeScript definitions
└── App.tsx             # Main application component
```

## Performance

- Bundle size target: <200KB gzipped
- Current build: ~74KB gzipped
- Code splitting enabled for optimal loading
- Mobile-first responsive design

## Technology Stack

- React 19 with new compiler
- React Router 6 for routing
- TypeScript for type safety
- Vite for fast development and optimized builds
- Native fetch API for HTTP requests
- CSS Modules for component-scoped styling