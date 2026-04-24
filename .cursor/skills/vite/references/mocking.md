# Vite Mocking Strategies

When to use: Testing Blade components, Livewire components, or routes that depend on Vite assets without running the actual build process.

## Patterns

**Mock Vite facade in Laravel tests**
```php
// tests/Feature/MarketingPageTest.php
use Illuminate\Support\Facades\Vite;

public function test_sales_page_renders_without_build(): void
{
    Vite::shouldReceive('asset')->andReturn('http://localhost:5173/resources/css/app.css');
    
    $response = $this->get('/');
    
    $response->assertOk();
    $response->assertSee('app.css');
}
```

**Mock import.meta.env in Vitest**
```javascript
// tests/setup.js
vi.stubGlobal('import.meta.env', {
    DEV: true,
    PROD: false,
    VITE_APP_URL: 'https://idx-api.quantyralabs.cc'
});

// In individual tests
test('uses env value', () => {
    vi.stubGlobal('import.meta.env', { VITE_APP_URL: 'https://test.example.com' });
    expect(getApiUrl()).toBe('https://test.example.com');
});
```

**Mock HMR socket for stability**
```javascript
// vitest.config.js
export default defineConfig({
    test: {
        environment: 'jsdom',
        globals: true,
        // Prevents actual WebSocket connections in tests
        setupFiles: ['./tests/setup-mocks.js']
    }
});

// tests/setup-mocks.js
vi.mock('vite/client', () => ({
    default: {
        on: vi.fn(),
        send: vi.fn()
    }
}));
```

## Warning

Mocking Vite's `import.meta.hot` can mask real HMR bugs—only mock at the module boundary, never inside component logic. Laravel's `Vite::fake()` facade method exists but behaves differently than full mocking; prefer `shouldReceive()` for precise assertions.