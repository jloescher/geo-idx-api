<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Attributes\Fillable;
use Illuminate\Database\Eloquent\Builder;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Illuminate\Support\Str;

#[Fillable([
    'email',
    'token_hash',
    'invited_by',
    'expires_at',
    'accepted_at',
])]
class UserInvitation extends Model
{
    /**
     * @return array<string, string>
     */
    protected function casts(): array
    {
        return [
            'expires_at' => 'datetime',
            'accepted_at' => 'datetime',
        ];
    }

    public function inviter(): BelongsTo
    {
        return $this->belongsTo(User::class, 'invited_by');
    }

    /**
     * @param  Builder<self>  $query
     * @return Builder<self>
     */
    public function scopeValid(Builder $query): Builder
    {
        return $query
            ->whereNull('accepted_at')
            ->where('expires_at', '>', now());
    }

    /**
     * @param  Builder<self>  $query
     * @return Builder<self>
     */
    public function scopeForEmail(Builder $query, string $email): Builder
    {
        return $query->where('email', mb_strtolower(trim($email)));
    }

    public static function hashPlainToken(string $plainToken): string
    {
        return hash('sha256', $plainToken);
    }

    /**
     * @return array{plain: string, model: self}
     */
    public static function createWithPlainToken(array $attributes): array
    {
        $plain = Str::random(64);

        $model = self::query()->create([
            ...$attributes,
            'email' => mb_strtolower(trim((string) $attributes['email'])),
            'token_hash' => self::hashPlainToken($plain),
        ]);

        return ['plain' => $plain, 'model' => $model];
    }

    public static function findValidByPlainToken(string $plainToken): ?self
    {
        $hash = self::hashPlainToken($plainToken);

        return self::query()
            ->where('token_hash', $hash)
            ->valid()
            ->first();
    }

    public function markAccepted(): void
    {
        $this->forceFill(['accepted_at' => now()])->save();
    }
}
