<?php

namespace Tests\Unit\AgentPortal;

use App\Services\AgentPortal\DefaultMultiMlsQueryCompiler;
use PHPUnit\Framework\TestCase;

class DefaultMultiMlsQueryCompilerTest extends TestCase
{
    public function test_compile_groups_filters_by_mls_scope(): void
    {
        $compiler = new DefaultMultiMlsQueryCompiler;
        $filters = [
            ['field' => 'property.list_price', 'operator' => 'gte', 'value' => 100_000],
        ];
        $scope = [
            ['mls_code' => 'stellar', 'dataset_code' => 'stellar'],
        ];

        $compiled = $compiler->compile($filters, $scope);

        $this->assertArrayHasKey('stellar@stellar', $compiled);
        $this->assertSame('stellar', $compiled['stellar@stellar']['mls_code']);
        $this->assertSame($filters, $compiled['stellar@stellar']['filters']);
    }

    public function test_validate_requires_mls_scope(): void
    {
        $compiler = new DefaultMultiMlsQueryCompiler;

        $errors = $compiler->validate([], []);

        $this->assertArrayHasKey('mls_scope', $errors);
    }
}
