/**
 * Minimal Alpine state for the subscriber dashboard (API-only product).
 *
 * @param {Record<string, unknown>} [_boot] Ignored; kept for PHPUnit / legacy boot payloads.
 */
export function createDashboardAlpineState(_boot = {}) {
    return {
        toast: '',
    };
}
