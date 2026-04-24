# Roadmap & Experiments

Use when planning feature rollouts, A/B testing configuration changes, or measuring adoption of new capabilities.

## Patterns

**Feature Flag via Config + Database**

```php
// Check eligibility before exposing new feature
public function isEligibleForFeature($locationId, $feature)
{
    // Config-based global enable
    if (!config("features.{$feature}.enabled")) {
        return false;
    }
    
    // Percentage rollout based on location ID hash
    $rolloutPercent = config("features.{$feature}.rollout_percent", 0);
    $hash = crc32($locationId) % 100;
    
    // Manual override in database
    $override = \DB::table('feature_flags')
        ->where('ghl_location_id', $locationId)
        ->where('feature', $feature)
        ->value('status');
    
    return $override === 'on' || ($override !== 'off' && $hash < $rolloutPercent);
}
```

**Experiment Result Tracking**

```sql
-- Compare conversion rates between widget theme variants
SELECT 
    wc.widget_theme as variant,
    COUNT(DISTINCT ql.ghl_location_id) as locations,
    COUNT(ql.id) as total_leads,
    ROUND(COUNT(ql.id)::numeric / COUNT(DISTINCT ql.ghl_location_id), 2) as leads_per_location
FROM ghl_widget_configs wc
LEFT JOIN quantyra_leads ql ON wc.ghl_location_id = ql.ghl_location_id
    AND ql.created_at > wc.updated_at  -- Leads after config change
WHERE wc.widget_theme IN ('default', 'minimal', 'premium')
    AND ql.created_at > NOW() - INTERVAL '14 days'
GROUP BY wc.widget_theme;
```

**GIS Layer Rollout Tracking**

```php
// Track which MLS markets use the parcel overlay
$gisAdoption = \DB::table('gis_cache')
    ->select('mls_code', \DB::raw('COUNT(DISTINCT query_hash) as unique_queries'))
    ->where('created_at', '>', now()->subDays(7))
    ->groupBy('mls_code')
    ->pluck('unique_queries', 'mls_code');
```

## Warning

Never use random assignment for revenue-critical experiments—location ID hash ensures consistent experiences but may bias toward certain geographic regions. Validate hash distribution across markets before interpreting results as causal.