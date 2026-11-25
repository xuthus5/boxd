import process from 'node:process';

export const E2E_PASSWORD = process.env.BOXD_E2E_PASSWORD || 'Boxd-E2E-2026!';
export const E2E_PORT = Number(process.env.BOXD_E2E_PORT || 17662);
export const E2E_BASE_URL = `http://127.0.0.1:${E2E_PORT}`;
