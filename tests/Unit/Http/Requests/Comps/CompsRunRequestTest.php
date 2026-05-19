<?php

namespace Tests\Unit\Http\Requests\Comps;

use App\Http\Requests\Comps\CompsRunRequest;
use Illuminate\Support\Facades\Validator;
use Tests\TestCase;

class CompsRunRequestTest extends TestCase
{
    public function test_merges_deprecated_stellar_subject_fields(): void
    {
        $request = CompsRunRequest::create('/api/v1/comps', 'POST', [
            'subject' => [
                'type' => 'off_market',
                'lat' => 27.95,
                'lng' => -82.45,
                'stellar_flood_zone_code' => 'X',
                'stellar_total_monthly_fees' => 500.22,
            ],
            'mode' => 'A',
            'scope' => [
                'type' => 'radius',
                'radius_miles' => 1,
            ],
        ]);

        $request->setContainer($this->app);
        $prepare = new \ReflectionMethod(CompsRunRequest::class, 'prepareForValidation');
        $prepare->invoke($request);

        $this->assertSame('X', $request->input('subject.flood_zone_code'));
        $this->assertSame(500.22, $request->input('subject.monthly_fees'));

        $validator = Validator::make(
            $request->all(),
            (new CompsRunRequest)->rules(),
        );

        $this->assertFalse($validator->fails());
    }
}
