import { defineConfig } from 'vite';
import laravel from 'laravel-vite-plugin';
import tailwindcss from '@tailwindcss/vite';

const appUrl = process.env.APP_URL ?? 'https://dev-idx.quantyralabs.cc';
const hmrHost = process.env.VITE_HMR_HOST ?? 'dev-idx.quantyralabs.cc';
const hmrProtocol = process.env.VITE_HMR_PROTOCOL ?? 'wss';
const hmrClientPort = Number(process.env.VITE_HMR_CLIENT_PORT ?? 443);

export default defineConfig({
    plugins: [
        laravel({
            input: [
                'resources/css/app.css',
                'resources/css/filament-dashboard.css',
                'resources/js/app.js',
            ],
            refresh: true,
        }),
        tailwindcss(),
    ],
    server: {
        host: '0.0.0.0',
        port: Number(process.env.VITE_PORT ?? 5173),
        strictPort: true,
        cors: {
            origin: [appUrl, 'https://idx.quantyralabs.cc', 'https://dev-idx.quantyralabs.cc', 'https://staging-idx.quantyralabs.cc'],
        },
        hmr: {
            host: hmrHost,
            protocol: hmrProtocol,
            clientPort: hmrClientPort,
        },
        watch: {
            ignored: ['**/storage/framework/views/**'],
        },
    },
});
