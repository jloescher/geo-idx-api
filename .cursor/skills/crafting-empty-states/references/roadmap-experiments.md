# Roadmap & Experiments

Use when A/B testing empty state copy, CTA placement, or feature availability for different subscription tiers.

## Patterns

### CTA Variant Testing
Test "Get started" vs "Configure widget" vs "Complete setup" on the `/leadconnector/installation-complete` empty state. The page controller randomly assigns variant via `session(['empty_state_variant' => rand(1, 3)])` and renders Blade partials `complete-variant-[n].blade.php`. Track completion rates per variant.

### Tiered Feature Teasers
For GIS parcel preview features, Smart tier users see actual parcel polygons on the map; Pro tier users see a blurred polygon overlay with "Upgrade to see parcel boundaries" tooltip. The empty state for Pro users includes a feature comparison grid with lock icons.

### Gradual Rollout Gates
New features like `PersistGisGeoJsonBackupJob` start with a config flag `GIS_BACKUP_ENABLED=false`. Beta users see the backup option in their dashboard; others see a "Coming soon" empty state with an email capture for early access. Roll out by moving user IDs into a feature flag array.

## Warning

Do not run experiments on critical compliance paths. The MLS domain registration flow and OAuth callback handling must remain consistent for all users. Experiment only on presentation layers (empty state copy, CTA buttons, visual hierarchy) and never on the underlying validation logic that enforces Stellar MLS rules.