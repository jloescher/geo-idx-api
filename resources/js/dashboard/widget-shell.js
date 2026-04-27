const dashboardWidgetPreviewStorageKey = 'quantyra_dashboard_widget_site_key';

/**
 * Dashboard widget library Alpine state (kept out of inline x-data to avoid
 * HTML attribute quote / parser issues when @js() emits double quotes).
 *
 * @param {{ previewApiKey: string, widgetValidateUrl: string, csrfToken: string, appUrl: string, widgetLoaderBaseUrl?: string }} boot
 */
export function createDashboardAlpineState(boot) {
    const appUrl = typeof boot.appUrl === 'string' ? boot.appUrl : '';
    const widgetLoaderBaseUrl =
        typeof boot.widgetLoaderBaseUrl === 'string' && boot.widgetLoaderBaseUrl !== ''
            ? boot.widgetLoaderBaseUrl
            : appUrl;
    const widgetValidateUrl = typeof boot.widgetValidateUrl === 'string' ? boot.widgetValidateUrl : '';
    const csrfToken = typeof boot.csrfToken === 'string' ? boot.csrfToken : '';
    const bootKey = typeof boot.previewApiKey === 'string' ? boot.previewApiKey.trim() : '';

    return {
        toast: '',
        previewWidget: '',
        previewLoading: false,
        previewError: '',
        previewApiKey: bootKey,
        widgetValidateUrl,
        csrfToken,
        appUrl,
        widgetLoaderBaseUrl,
        init() {
            if (this.previewApiKey === '' && typeof localStorage !== 'undefined') {
                const stored = localStorage.getItem(dashboardWidgetPreviewStorageKey);
                if (stored !== null && String(stored).trim() !== '') {
                    this.previewApiKey = String(stored).trim();
                }
            }
            this.$watch('previewApiKey', (value) => {
                if (typeof localStorage === 'undefined') {
                    return;
                }
                const trimmed = String(value ?? '').trim();
                if (trimmed.length >= 8) {
                    localStorage.setItem(dashboardWidgetPreviewStorageKey, trimmed);
                } else if (trimmed === '') {
                    localStorage.removeItem(dashboardWidgetPreviewStorageKey);
                }
            });
        },
        resolveWidgetType(slug) {
            const map = {
                'search-bar': 'search',
                'listing-cards': 'community',
                'property-detail': 'property',
                'map-search': 'map',
            };

            return map[slug] || 'search';
        },
        async validatePreviewContext() {
            if (! String(this.previewApiKey ?? '').trim()) {
                throw new Error(
                    'Add a widget site key to preview. Reload the dashboard after your plan activates to auto-create one, '
                        + 'or paste your key (saved in this browser). Approved domains must include the site you embed on.',
                );
            }
            const res = await fetch(this.widgetValidateUrl, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    Accept: 'application/json',
                    'X-CSRF-TOKEN': this.csrfToken,
                    'X-Requested-With': 'XMLHttpRequest',
                },
                credentials: 'same-origin',
                body: JSON.stringify({
                    token: this.previewApiKey,
                    hostname: window.location.hostname,
                    referrer: document.referrer || null,
                    requireFooter: true,
                }),
            });
            const data = await res.json().catch(() => ({}));
            if (! res.ok || ! data.ok) {
                throw new Error(data.message || 'Preview host validation failed.');
            }
        },
        ensureLoaderScript() {
            return new Promise((resolve, reject) => {
                if (window.QuantyraGeoIDX) {
                    resolve();

                    return;
                }
                const existing = document.getElementById('dashboard-widget-loader');
                if (existing) {
                    existing.addEventListener('load', () => resolve(), { once: true });
                    existing.addEventListener('error', () => reject(new Error('Failed to load widget runtime.')), { once: true });

                    return;
                }
                const script = document.createElement('scr' + 'ipt');
                script.id = 'dashboard-widget-loader';
                script.src = `${this.widgetLoaderBaseUrl}/widget/loader.js?token=${encodeURIComponent(this.previewApiKey)}&primaryColor=3b82f6&accentColor=10b981`;
                script.setAttribute('data-footer-required', 'true');
                script.async = true;
                script.addEventListener('load', () => resolve(), { once: true });
                script.addEventListener('error', () => reject(new Error('Failed to load widget runtime.')), { once: true });
                document.body.appendChild(script);
            });
        },
        async mountPreview() {
            this.previewError = '';
            if (! this.previewWidget) {
                return;
            }
            this.previewLoading = true;
            try {
                await this.validatePreviewContext();
                await this.ensureLoaderScript();
                const type = this.resolveWidgetType(this.previewWidget);
                const mountNode = this.$refs.previewCanvas;
                mountNode.innerHTML = '';
                await window.QuantyraGeoIDX.initWidget(type, { target: mountNode });
            } catch (error) {
                const msg = error instanceof TypeError && String(error.message || '').includes('fetch')
                    ? 'Network error while loading the preview (often a blocked request to the IDX API or a CORS issue). Confirm IDX_API_PUBLIC_URL matches the API host and try again.'
                    : (error.message || 'Unable to render preview.');
                this.previewError = msg;
            } finally {
                this.previewLoading = false;
            }
        },
        async openPreview(slug) {
            this.previewWidget = slug;
            if (! String(this.previewApiKey ?? '').trim() && typeof localStorage !== 'undefined') {
                const stored = localStorage.getItem(dashboardWidgetPreviewStorageKey);
                if (stored !== null && String(stored).trim() !== '') {
                    this.previewApiKey = String(stored).trim();
                }
            }
            await this.$nextTick();
            await this.mountPreview();
        },
        copyEmbed(slug) {
            const openTag = '<scr' + 'ipt';
            const closeTag = '<' + '/scr' + 'ipt>';
            const code = openTag + ` src='${this.widgetLoaderBaseUrl}/widget/loader.js?token=YOUR_SITE_KEY&primaryColor=3b82f6&accentColor=10b981' data-widget='${slug}' data-footer-required='true' async` + '>' + closeTag;
            navigator.clipboard.writeText(code);
            this.toast = 'Copied embed code!';
            setTimeout(() => this.toast = '', 2200);
        },
    };
}
