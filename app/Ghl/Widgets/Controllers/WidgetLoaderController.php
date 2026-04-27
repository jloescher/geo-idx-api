<?php

namespace App\Ghl\Widgets\Controllers;

use Illuminate\Filesystem\Filesystem;
use Illuminate\Http\Response;

/**
 * Revenue Impact: One script tag → IDX surfaces on GHL sites → more inventory views → more gated leads.
 */
class WidgetLoaderController
{
    public function __construct(
        private readonly Filesystem $filesystem,
    ) {}

    public function __invoke(): Response
    {
        $productionPath = public_path('js/widgets/prod/loader.js');
        $developmentPath = base_path('dist/widgets/loader.js');

        $scriptBody = $this->filesystem->exists($productionPath)
            ? $this->filesystem->get($productionPath)
            : ($this->filesystem->exists($developmentPath) ? $this->filesystem->get($developmentPath) : 'console.error("Widget loader build not found. Run npm run build:widgets.");');

        return response($scriptBody, 200, [
            'Content-Type' => 'application/javascript; charset=UTF-8',
            'Cache-Control' => 'public, max-age=86400, s-maxage=86400',
        ]);
    }
}
