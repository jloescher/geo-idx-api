<?php

namespace App\Services\AgentPortal\Contracts;

interface FieldMappingAdapterInterface
{
    public function mlsCode(): string;

    public function datasetCode(): string;

    public function toSourceField(string $canonicalFieldKey): ?string;

    public function toCanonicalField(string $sourceFieldKey): ?string;

    /**
     * @return array<string, mixed>
     */
    public function transformIn(string $canonicalFieldKey, mixed $value, string $operator): array;

    public function transformOut(string $sourceFieldKey, mixed $value): mixed;
}
