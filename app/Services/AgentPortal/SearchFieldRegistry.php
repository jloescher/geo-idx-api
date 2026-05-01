<?php

namespace App\Services\AgentPortal;

/**
 * Schema-driven registry for agent search UI and query compilation (canonical keys).
 *
 * @phpstan-type FieldDef array{
 *     key: string,
 *     label: string,
 *     category: string,
 *     type: string,
 *     operators: list<string>
 * }
 */
final class SearchFieldRegistry
{
    /**
     * @return list<FieldDef>
     */
    public function coreFields(): array
    {
        return [
            [
                'key' => 'property.list_price',
                'label' => 'List price',
                'category' => 'general',
                'type' => 'number',
                'operators' => ['between', 'gte', 'lte'],
            ],
            [
                'key' => 'property.bedrooms_total',
                'label' => 'Bedrooms',
                'category' => 'general',
                'type' => 'number',
                'operators' => ['between', 'gte', 'lte', 'eq'],
            ],
            [
                'key' => 'property.bathrooms_total',
                'label' => 'Bathrooms',
                'category' => 'general',
                'type' => 'number',
                'operators' => ['between', 'gte', 'lte', 'eq'],
            ],
            [
                'key' => 'location.city',
                'label' => 'City',
                'category' => 'locations',
                'type' => 'string',
                'operators' => ['in', 'contains', 'eq'],
            ],
            [
                'key' => 'listing.status',
                'label' => 'Status',
                'category' => 'general',
                'type' => 'enum',
                'operators' => ['in', 'eq'],
            ],
        ];
    }

    /**
     * @return array<string, FieldDef>
     */
    public function keyed(): array
    {
        $map = [];
        foreach ($this->coreFields() as $field) {
            $map[$field['key']] = $field;
        }

        return $map;
    }
}
