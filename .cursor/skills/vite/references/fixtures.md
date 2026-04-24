# Vite Test Fixtures

When to use: Providing sample CSS, JS, or manifest files for tests that verify asset processing, URL rewriting, or build output parsing.

## Patterns

**Minimal CSS fixture for parser tests**
```css
/* tests/fixtures/sample-tailwind.css */
@import "tailwindcss";

@layer components {
    .btn-primary {
        @apply bg-blue-600 text-white px-4 py-2 rounded;
    }
}
```

**Sample manifest fixture**
```json
// tests/fixtures/vite-manifest.json
{
    "resources/css/app.css": {
        "file": "assets/app-CQJy1v3x.css",
        "src": "resources/css/app.css",
        "isEntry": true
    },
    "resources/js/app.js": {
        "file": "assets/app-DQJy2v4y.js",
        "src": "resources/js/app.js",
        "isEntry": true,
        "css": ["assets/app-CQJy1v3x.css"]
    }
}
```

**Use fixture in test**
```php
public function test_image_url_rewriter_handles_built_assets(): void
{
    // Copy fixture to public/build for test isolation
    File::copyDirectory(base_path('tests/fixtures/build'), public_path('build'));
    
    $rewriter = new BridgeImageUrlRewriter();
    $json = $rewriter->rewrite($apiResponse);
    
    // Assert URLs point to built assets, not dev server
    $this->assertStringNotContainsString(':5173', $json);
}
```

## Warning

Never commit actual `public/build/` test artifacts to version control—use `.gitignore` and generate fixtures dynamically in `setUp()`. Fixtures must match the Tailwind 4 `@import "tailwindcss"` syntax (not the older `@tailwind` directives) or tests will fail when the real build runs.