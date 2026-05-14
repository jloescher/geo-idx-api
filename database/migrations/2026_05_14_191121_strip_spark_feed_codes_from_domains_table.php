<?php

use App\Models\Domain;
use Illuminate\Database\Migrations\Migration;

return new class extends Migration
{
    /**
     * Strip legacy Spark catalog codes from domain MLS allowlists and defaults.
     */
    public function up(): void
    {
        $bridgeDatasets = config('bridge.datasets', ['stellar']);
        if (! is_array($bridgeDatasets) || $bridgeDatasets === []) {
            $bridgeDatasets = ['stellar'];
        }

        $defaultCatalogKeys = [];
        foreach ($bridgeDatasets as $dataset) {
            if (! is_string($dataset)) {
                continue;
            }
            $dataset = trim($dataset);
            if ($dataset === '') {
                continue;
            }
            $defaultCatalogKeys[] = 'bridge_'.$dataset;
        }

        if ($defaultCatalogKeys === []) {
            $defaultCatalogKeys = ['bridge_stellar'];
        }

        $defaultWire = str_starts_with($defaultCatalogKeys[0], 'bridge_')
            ? substr($defaultCatalogKeys[0], strlen('bridge_'))
            : $defaultCatalogKeys[0];

        $legacy = ['space_coast', 'beaches'];

        Domain::query()->chunkById(100, function ($domains) use ($defaultCatalogKeys, $defaultWire, $legacy): void {
            foreach ($domains as $domain) {
                $dirty = false;

                $allowed = $domain->allowed_mls_datasets;
                if (is_array($allowed)) {
                    $filtered = [];
                    foreach ($allowed as $item) {
                        if (! is_string($item) || trim($item) === '') {
                            continue;
                        }
                        if (in_array(strtolower(trim($item)), $legacy, true)) {
                            continue;
                        }
                        $filtered[] = $item;
                    }

                    if ($filtered === []) {
                        $filtered = $defaultCatalogKeys;
                    }

                    if ($filtered !== $allowed) {
                        $domain->allowed_mls_datasets = $filtered;
                        $dirty = true;
                    }
                }

                $mlsDataset = $domain->mls_dataset;
                if (is_string($mlsDataset) && trim($mlsDataset) !== '') {
                    if (in_array(strtolower(trim($mlsDataset)), $legacy, true)) {
                        $domain->mls_dataset = $defaultWire;
                        $dirty = true;
                    }
                }

                if ($dirty) {
                    $domain->save();
                }
            }
        });
    }

    /**
     * Irreversible: Spark MLS feed codes were removed from the application catalog.
     */
    public function down(): void {}
};
