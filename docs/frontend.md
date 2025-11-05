# Frontend Development Guide

The Quickly Pick frontend is a minimal, mobile-first React application built with Vite and TypeScript.

## Quick Start

```bash
cd client-web
npm install
cp .env.example .env
npm run dev
```

The app will be available at `http://localhost:5173`

## Prerequisites

- **Node.js 18+** - [Install Node.js](https://nodejs.org/)
- **npm** (comes with Node.js)
- **Backend server** running at `http://localhost:3318`

## Installation

### 1. Install Dependencies

```bash
cd client-web
npm install
```

### 2. Configure Environment

```bash
cp .env.example .env
```

Edit `.env`:

```env
VITE_API_BASE_URL=http://localhost:3318
```

### 3. Start Development Server

```bash
npm run dev
```

The development server will start at `http://localhost:5173` with hot module replacement (HMR) enabled.

## Available Scripts

```bash
# Start development server with HMR
npm run dev

# Build for production
npm run build

# Preview production build locally
npm run preview

# Run linter
npm run lint

# Type check
npm run type-check
```

## Project Structure

```
client-web/
├── src/
│   ├── api/                    # API client and utilities
│   │   ├── client.ts          # API client with error handling
│   │   ├── config.ts          # API configuration
│   │   ├── utils.ts           # High-level API functions
│   │   └── index.ts           # Re-exports
│   ├── components/
│   │   ├── common/            # Reusable UI components
│   │   │   ├── Button.tsx
│   │   │   ├── Input.tsx
│   │   │   ├── Slider.tsx
│   │   │   ├── Card.tsx
│   │   │   ├── Container.tsx
│   │   │   ├── LoadingSpinner.tsx
│   │   │   └── ErrorBoundary.tsx
│   │   ├── poll/              # Poll-specific components
│   │   └── layout/            # Layout components
│   ├── pages/                 # Route components
│   │   ├── HomePage.tsx       # Landing page
│   │   ├── CreatePollPage.tsx # Poll creation wizard
│   │   ├── AdminPage.tsx      # Poll administration
│   │   ├── VotePage.tsx       # Voting interface
│   │   └── ResultsPage.tsx    # Results display
│   ├── types/
│   │   └── index.ts           # TypeScript type definitions
│   ├── App.tsx                # Main application component
│   ├── main.tsx               # Application entry point
│   └── index.css              # Global styles
├── public/
│   └── _redirects             # SPA routing support
├── dist/                      # Production build output
├── .env.example               # Environment variables template
├── vite.config.ts            # Vite configuration
├── tsconfig.json             # TypeScript configuration
└── package.json              # Dependencies and scripts
```

## Key Features

### Mobile-First Design

- Responsive layouts that work on all screen sizes
- Touch-friendly UI elements (48px minimum touch targets)
- Mobile-optimized typography and spacing

### Performance

- Code splitting for optimal loading
- Lazy loading of route components
- Optimized bundle size (~74KB gzipped)
- No external CSS frameworks

### User Experience

- Loading states and error boundaries
- Client-side form validation
- LocalStorage for session persistence
- Smooth transitions and interactions

## Development Workflow

### Adding a New Page

1. Create component in `src/pages/`:

```tsx
// src/pages/NewPage.tsx
import { Container, Card } from '../components/common'

export const NewPage = () => {
  return (
    <Container>
      <Card>
        <h1>New Page</h1>
        <p>Content goes here</p>
      </Card>
    </Container>
  )
}
```

2. Add route in `App.tsx`:

```tsx
import { NewPage } from './pages/NewPage'

// Inside Routes component:
<Route path="/new" element={<NewPage />} />
```

### Adding a New Component

Create component in appropriate directory:

```tsx
// src/components/common/MyComponent.tsx
import './MyComponent.css'

interface MyComponentProps {
  title: string
  onAction: () => void
}

export const MyComponent = ({ title, onAction }: MyComponentProps) => {
  return (
    <div className="my-component">
      <h3>{title}</h3>
      <button onClick={onAction}>Click me</button>
    </div>
  )
}
```

Export from index:

```tsx
// src/components/common/index.ts
export { MyComponent } from './MyComponent'
```

### API Integration

The API client is located in `src/api/`:

```tsx
import { apiClient, votingApi } from '../api'

// Direct API calls
const poll = await apiClient.getPoll(slug)
const results = await apiClient.getResults(slug)

// High-level convenience functions
await votingApi.claimUsername(slug, username)
await votingApi.submitBallot(slug, ratings)
```

### State Management

Use React's built-in state management:

```tsx
import { useState, useEffect } from 'react'

const [data, setData] = useState<DataType | null>(null)
const [loading, setLoading] = useState(true)
const [error, setError] = useState<string | null>(null)

useEffect(() => {
  const loadData = async () => {
    try {
      setLoading(true)
      const result = await apiClient.getData()
      setData(result)
    } catch (err) {
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }
  
  loadData()
}, [dependency])
```

### LocalStorage Persistence

Use the storage utilities:

```tsx
import { storage } from '../api/utils'

// Store data
storage.setAdminKey(pollId, adminKey)
storage.setVoterToken(slug, voterToken)

// Retrieve data
const adminKey = storage.getAdminKey(pollId)
const voterToken = storage.getVoterToken(slug)

// Clear data
storage.clearPollData(pollId)
storage.clearVoterData(slug)
```

## Styling Guidelines

### CSS Organization

- Component-specific styles in `.css` files next to components
- Global styles in `index.css`
- Use CSS custom properties for theming
- Follow mobile-first responsive design

### Design Principles

- **Minimal** - Border-only cards, no shadows
- **Clean** - Plenty of whitespace
- **Accessible** - High contrast, large touch targets
- **Fast** - No heavy CSS frameworks

### Example Component Styles

```css
/* Button.css */
.button {
  /* Touch target requirements */
  min-height: 48px;
  min-width: 120px;
  
  /* Spacing */
  padding: 16px 24px;
  
  /* Typography */
  font-size: 18px;
  
  /* Styling */
  border: 2px solid;
  background: transparent;
  cursor: pointer;
  
  /* Transitions */
  transition: all 0.2s ease;
}

.button:hover:not(:disabled) {
  background-color: rgba(0, 0, 0, 0.05);
}
```

## TypeScript Usage

### Type Definitions

All types are in `src/types/index.ts`:

```typescript
// API response types
export interface Poll {
  id: string
  title: string
  description: string
  creator_name: string
  status: 'draft' | 'open' | 'closed'
  slug: string
  created_at: string
  closed_at?: string
}

// Component prop types
export interface ButtonProps {
  children: React.ReactNode
  onClick?: () => void
  type?: 'button' | 'submit'
  disabled?: boolean
  fullWidth?: boolean
}
```

### Type Safety

Use TypeScript features for safer code:

```typescript
// Type guards
function isPoll(data: unknown): data is Poll {
  return typeof data === 'object' && 
    data !== null && 
    'id' in data && 
    'title' in data
}

// Generic state type
interface ComponentState<T = unknown> {
  loading: boolean
  error: string | null
  data: T | null
}
```

## Error Handling

### API Errors

```typescript
import { ApiError } from '../types'

try {
  const data = await apiClient.getData()
} catch (error) {
  if (error instanceof ApiError) {
    console.error(`API Error ${error.status}: ${error.message}`)
  } else {
    console.error('Network error:', error)
  }
}
```

### Error Boundaries

Wrap components with ErrorBoundary:

```tsx
import { ErrorBoundary } from './components/common'

<ErrorBoundary>
  <YourComponent />
</ErrorBoundary>
```

## Testing

### Manual Testing Checklist

- [ ] Test on mobile viewport (375px width)
- [ ] Test on tablet (768px width)  
- [ ] Test on desktop (1024px+ width)
- [ ] Test all form validations
- [ ] Test error states (network failures)
- [ ] Test loading states
- [ ] Verify API integration with backend
- [ ] Check LocalStorage persistence
- [ ] Test browser back/forward navigation

### Testing with Different API States

```bash
# Test with backend running
npm run dev

# Test with backend down (error handling)
# Stop backend, try creating/voting on polls

# Test with slow network
# Use browser DevTools to throttle network
```

## Building for Production

### Production Build

```bash
npm run build
```

Output will be in `dist/` directory:

```
dist/
├── assets/
│   ├── index-[hash].js
│   ├── vendor-[hash].js
│   └── index-[hash].css
├── _redirects           # SPA routing support
└── index.html
```

### Preview Production Build

```bash
npm run preview
```

### Build Optimization

The build is optimized for:

- **Code splitting** - Vendor code separated
- **Tree shaking** - Unused code removed
- **Minification** - JS and CSS minified
- **Hashing** - Cache busting for assets

## Deployment

### Static Hosting (Netlify, Vercel, etc.)

1. Build the app:
```bash
npm run build
```

2. Deploy `dist/` directory

3. Configure API URL:
```env
VITE_API_BASE_URL=https://api.yourdomain.com
```

The `_redirects` file in `dist/` ensures SPA routing works correctly.

### Environment Variables

Set these in your deployment platform:

```env
VITE_API_BASE_URL=https://api.production.com
```

## Troubleshooting

### Common Issues

#### "Failed to fetch" errors

- Check backend is running at correct URL
- Verify `VITE_API_BASE_URL` in `.env`
- Check for CORS issues in backend

#### Routes not working after refresh

- Ensure `_redirects` file is in `dist/`
- Configure hosting platform for SPA routing

#### Styles not loading

- Clear browser cache
- Rebuild app: `npm run build`
- Check for CSS import errors

#### TypeScript errors

- Run type check: `npm run type-check`
- Ensure all dependencies are installed
- Check `tsconfig.json` configuration

### Debug Mode

Enable detailed logging:

```typescript
// src/api/config.ts
export const DEBUG = import.meta.env.DEV

// Use in code
if (DEBUG) {
  console.log('API request:', endpoint, data)
}
```

## Performance Tips

### Bundle Size

Monitor bundle size:

```bash
npm run build
# Check output for bundle sizes
```

Keep total bundle under 200KB gzipped.

### Loading Performance

- Use lazy loading for routes
- Implement code splitting
- Optimize images (use WebP format)
- Minimize third-party dependencies

### Runtime Performance

- Avoid unnecessary re-renders
- Use React DevTools Profiler
- Implement proper key props for lists
- Memoize expensive computations

## Best Practices

### Component Design

- Keep components small and focused
- Use composition over inheritance
- Extract reusable logic into hooks
- Maintain consistent prop interfaces

### Code Organization

- Group related files together
- Use index files for clean exports
- Keep API logic separate from UI
- Use TypeScript for type safety

### Accessibility

- Use semantic HTML elements
- Add ARIA labels where needed
- Ensure keyboard navigation works
- Maintain sufficient color contrast

## Resources

### Documentation

- [React Documentation](https://react.dev/)
- [Vite Guide](https://vitejs.dev/guide/)
- [TypeScript Handbook](https://www.typescriptlang.org/docs/)
- [React Router](https://reactrouter.com/)

### Tools

- [React DevTools](https://react.dev/learn/react-developer-tools)
- [TypeScript Playground](https://www.typescriptlang.org/play)
- [Can I Use](https://caniuse.com/) - Browser compatibility

---

**Need help?** Check the API documentation at [docs/api.md](api.md) or create an issue in the repository.