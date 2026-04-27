import { createDashboardAlpineState } from './dashboard/widget-shell.js';

window.__createDashboardAlpineState = createDashboardAlpineState;

const analyticsRoot = document;

analyticsRoot.addEventListener('click', (event) => {
    const target = event.target;

    if (!(target instanceof Element)) {
        return;
    }

    const trackedElement = target.closest('[data-event-name]');

    if (!(trackedElement instanceof HTMLElement)) {
        return;
    }

    const eventName = trackedElement.dataset.eventName;

    if (!eventName) {
        return;
    }

    window.dataLayer = window.dataLayer || [];
    window.dataLayer.push({
        event: eventName,
        source: 'sales_landing_page',
    });
});
