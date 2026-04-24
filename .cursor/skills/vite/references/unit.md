# Vite Unit Testing

When to use: Testing frontend utility functions, CSS class utilities, or Vite configuration helpers in isolation without full build process.

## Patterns

**Test a Tailwind class utility**
```javascript
// resources/js/utils/classNames.js
export function mergeClasses(base, overrides) {
    return [...new Set([...base.split(' '), ...overrides.split(' ')])].join(' ');
}

// tests/unit/classNames.test.js
import { test, expect } from 'vitest';
import { mergeClasses } from '@/utils/classNames';

test('merges Tailwind classes without duplicates', () => {
    const result = mergeClasses('bg-white text-gray-900', 'bg-gray-50');
    expect(result).toBe('bg-white text-gray-900 bg-gray-50');
});
```

**Test Vite environment detection**
```javascript
// resources/js/utils/env.js
export const isDev = () => import.meta.env.DEV === true;

// tests/unit/env.test.js
import { test, expect, vi } from 'vitest';
import { isDev } from '@/utils/env';

test('detects development mode from import.meta.env', () => {
    vi.stubGlobal('import.meta.env', { DEV: true });
    expect(isDev()).toBe(true);
});
```

**Test HMR host configuration**
```php
// tests/Unit/ViteConfigTest.php
public function test_vite_config_allows_cloudflare_tunnel_hosts(): void
{
    $config = file_get_contents(base_path('vite.config.js'));
    
    $this->assertStringContainsString("host: ['localhost', '127.0.0.1', '.trycloudflare.com']", $config);
    $this->assertStringContainsString('strictPort: false', $config);
}
```

## Warning

Unit tests for Vite configs should not actually start the Vite server—parse config files as strings or test helper functions in isolation. Avoid testing build output in unit tests; use integration tests for compiled asset verification.