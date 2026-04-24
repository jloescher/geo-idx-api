# Vite Integration Testing

When to use: Testing the full build pipeline, asset compilation, Laravel Vite plugin integration, or Blade template rendering with Vite directives.

## Patterns

**Test Vite assets are compiled**
```php
// tests/Feature/ViteBuildTest.php
public function test_vite_build_generates_manifest(): void
{
    $this->artisan('vite:build')->assertSuccessful();
    
    $manifest = public_path('build/manifest.json');
    $this->assertFileExists($manifest);
    
    $contents = json_decode(file_get_contents($manifest), true);
    $this->assertArrayHasKey('resources/css/app.css', $contents);
    $this->assertArrayHasKey('resources/js/app.js', $contents);
}
```

**Test Blade renders with Vite directive**
```php
public function test_marketing_page_includes_vite_assets(): void
{
    $response = $this->get(route('marketing.sales'));
    
    $response->assertOk();
    $response->assertSee('resources/css/app.css', escape: false);
    $response->assertSee('resources/js/app.js', escape: false);
}
```

**Test Tailwind 4 processes correctly**
```php
public function test_tailwind_classes_are_compiled(): void
{
    $css = file_get_contents(resource_path('css/app.css'));
    
    // Verify Tailwind 4 @import syntax (no tailwind.config.js needed)
    $this->assertStringContainsString('@import "tailwindcss"', $css);
    
    // Custom layer directives present
    $this->assertStringContainsString('@layer components', $css);
}
```

## Warning

Integration tests that trigger `vite:build` are slow—group them in a dedicated test suite and skip in CI unless explicitly running build verification. The `npm run build` process can timeout in resource-constrained environments; set `VITE_BUILD_TIMEOUT` if your test runner supports it.