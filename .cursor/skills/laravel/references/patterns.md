# Laravel Patterns in IDX API

When to use: Implementing new features, creating controllers, models, or services in the Quantyra IDX API codebase.

## Import-Style Routing

Use explicit `use` statements with class references instead of string-based controller references.

```php
// routes/api.php
use App\Http\Controllers\Api\BridgeProxyController;

Route::get('/v1/listings', [BridgeProxyController::class, 'listings'])
    ->name('api.listings')
    ->middleware('domain.token');
```

## Constructor Property Promotion with Readonly Services

Inject services via constructor with readonly properties and private visibility.

```php
class BridgeProxyController extends Controller
{
    public function __construct(
        private readonly BridgeHttpService $bridgeService,
        private readonly ListingsCacheService $cacheService,
    ) {}

    public function listings(Request $request): JsonResponse
    {
        $listings = $this->cacheService->get($request);
        // ...
    }
}
```

## PHP 8 Attributes for Models

Use attributes instead of `$fillable`/`$hidden` properties; use `casts()` method.

```php
use Illuminate\Database\Eloquent\Attributes\Fillable;
use Illuminate\Database\Eloquent\Attributes\Hidden;

#[Fillable(['domain_slug', 'is_active'])]
#[Hidden(['compressed_data'])]
class ListingsCache extends Model
{
    protected function casts(): array
    {
        return [
            'last_updated' => 'datetime',
            'is_active' => 'boolean',
        ];
    }
}
```

## ⚠️ Pitfall: Config Caching with Nested Config Calls

Never use `config()` inside config files. Use `env()` directly to avoid cache-breaking issues.

```php
// config/ghl.php - CORRECT
'redirect_uri' => env('GHL_REDIRECT_URI', env('APP_URL') . '/oauth/leadconnector/callback'),

// WRONG - causes cache issues
'redirect_uri' => config('app.url') . '/oauth/callback',