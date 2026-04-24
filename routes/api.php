<?php

use App\Ghl\Http\Controllers\GhlApiController;
use App\Ghl\Http\Middleware\AuthenticateGhlLocation;
use App\Http\Controllers\Api\BridgeProxyController;
use App\Http\Controllers\GisProxyController;
use App\Models\User;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\Hash;
use Illuminate\Support\Facades\Route;
use Illuminate\Validation\ValidationException;

Route::post('/auth/token', function (Request $request): array {
    $credentials = $request->validate([
        'email' => ['required', 'email'],
        'password' => ['required', 'string'],
        'device_name' => ['required', 'string', 'max:255'],
    ]);

    $user = User::query()->where('email', $credentials['email'])->first();

    if (! $user || ! Hash::check($credentials['password'], $user->password)) {
        throw ValidationException::withMessages([
            'email' => ['The provided credentials are incorrect.'],
        ]);
    }

    return [
        'token' => $user->createToken($credentials['device_name'])->plainTextToken,
    ];
});

Route::middleware('auth:sanctum')->group(function (): void {
    Route::get('/auth/user', fn (Request $request): ?User => $request->user());
});

Route::prefix('leadconnector')->middleware([AuthenticateGhlLocation::class])->group(function () {
    Route::get('/leads', [GhlApiController::class, 'leads']);
    Route::get('/leads/{id}', [GhlApiController::class, 'lead']);
    Route::get('/subscriptions', [GhlApiController::class, 'subscriptions']);
    Route::get('/stats', [GhlApiController::class, 'stats']);
    Route::get('/config', [GhlApiController::class, 'config']);
});

Route::prefix('v1')->middleware(['domain.token'])->group(function () {
    Route::get('/gis', [GisProxyController::class, 'show']);
    Route::get('/mls/{mlsCode}/gis', [GisProxyController::class, 'showForMls'])->where('mlsCode', '[A-Za-z0-9_\-]+');

    Route::get('/listings', [BridgeProxyController::class, 'listings']);
    Route::get('/listings/{listingId}', [BridgeProxyController::class, 'listing'])->where('listingId', '[^/]+');

    Route::get('/agents', [BridgeProxyController::class, 'agents']);
    Route::get('/agents/{agentId}', [BridgeProxyController::class, 'agent'])->where('agentId', '[^/]+');

    Route::get('/offices', [BridgeProxyController::class, 'offices']);
    Route::get('/offices/{officeId}', [BridgeProxyController::class, 'office'])->where('officeId', '[^/]+');

    Route::get('/openhouses', [BridgeProxyController::class, 'openHouses']);
    Route::get('/openhouses/{openhouseId}', [BridgeProxyController::class, 'openHouse'])->where('openhouseId', '[^/]+');

    Route::get('/properties', [BridgeProxyController::class, 'properties']);
    Route::get('/properties/{listingKey}', [BridgeProxyController::class, 'property'])->where('listingKey', '[^/]+');

    Route::get('/members', [BridgeProxyController::class, 'members']);
    Route::get('/members/{memberKey}', [BridgeProxyController::class, 'member'])->where('memberKey', '[^/]+');

    Route::get('/reso-offices', [BridgeProxyController::class, 'resoOffices']);
    Route::get('/reso-offices/{officeKey}', [BridgeProxyController::class, 'resoOffice'])->where('officeKey', '[^/]+');

    Route::get('/reso-openhouses', [BridgeProxyController::class, 'resoOpenHouses']);
    Route::get('/reso-openhouses/{openHouseKey}', [BridgeProxyController::class, 'resoOpenHouse'])->where('openHouseKey', '[^/]+');

    Route::get('/lookup', [BridgeProxyController::class, 'lookup']);

    Route::prefix('pub')->group(function () {
        Route::get('/parcels', [BridgeProxyController::class, 'pubParcels']);
        Route::get('/parcels/{parcelId}', [BridgeProxyController::class, 'pubParcel'])->where('parcelId', '[^/]+');
        Route::get('/parcels/{parcelId}/assessments', [BridgeProxyController::class, 'pubParcelAssessments'])->where('parcelId', '[^/]+');
        Route::get('/parcels/{parcelId}/transactions', [BridgeProxyController::class, 'pubParcelTransactions'])->where('parcelId', '[^/]+');
        Route::get('/assessments', [BridgeProxyController::class, 'pubAssessments']);
        Route::get('/transactions', [BridgeProxyController::class, 'pubTransactions']);
    });

    Route::get('/zestimates', [BridgeProxyController::class, 'zestimates']);

    Route::prefix('zgecon')->group(function () {
        Route::get('/marketreport', [BridgeProxyController::class, 'zgeconMarketReport']);
        Route::get('/marketreport/replication', [BridgeProxyController::class, 'zgeconMarketReportReplication']);
        Route::get('/region', [BridgeProxyController::class, 'zgeconRegion']);
        Route::get('/cut', [BridgeProxyController::class, 'zgeconCut']);
        Route::get('/type', [BridgeProxyController::class, 'zgeconType']);
    });

    Route::prefix('reviews')->group(function () {
        Route::get('/reviews', [BridgeProxyController::class, 'reviews']);
        Route::get('/reviewees', [BridgeProxyController::class, 'reviewees']);
    });
});
