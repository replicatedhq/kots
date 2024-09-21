import { test, expect } from '@playwright/test';
import { login, uploadLicense } from '../shared';

const CUSTOMER_ID = '2k6jemHbYgZFqtwgyjqiVfjRQqi';
const APP_ID = '2k6j65t0STtrZ1emyP5PUqBIQ23';
const AUTOMATED_CHANNEL_ID = '2k6j62KPRQLjO0tF9zZB6zJJukg';
const ALTERNATE_CHANNEL_ID = '2k6j61j49IPyDyQlbmRZJsxy3TP';

test('change channel', async ({ page }) => {
    test.slow();
    await changeChannel(AUTOMATED_CHANNEL_ID);
    await login(page);
    await uploadLicense(page, expect);
    await page.getByRole('button', { name: 'Deploy' }).click();
    await expect(page.locator('#app')).toContainText('Automated');
    await changeChannel(ALTERNATE_CHANNEL_ID);

    await page.getByText('Sync license').click();

    await expect(page.getByLabel('Next step')).toContainText('License synced', { timeout: 10000 });
    await page.getByRole('button', { name: 'Ok, got it!' }).click();

    await expect(page.locator('#app')).toContainText('Alternate');
    await expect(page.locator('#app')).toContainText('v1.0.1', { timeout: 10000 });
    await expect(page.locator('#app')).toContainText('Upstream Update', { timeout: 10000 });

    await page.getByRole('button', { name: 'Deploy', exact: true }).click();
    await page.getByRole('button', { name: 'Yes, Deploy' }).click();

    await expect(page.locator('#app')).toContainText('Currently deployed version', { timeout: 15000 });
    await expect(page.getByText('v1.0.0')).not.toBeVisible();
    await expect(page.getByText('v1.0.1')).toBeVisible();
});

async function changeChannel(channelId: string) {
    await fetch(`https://api.replicated.com/vendor/v3/customer/${CUSTOMER_ID}`, {
        method: 'PUT',
        headers: {
            'Content-Type': 'application/json',
            'Authorization': process.env.REPLICATED_API_TOKEN,
        },
        body: JSON.stringify({
            "app_id": APP_ID,
            "name": "github-action", // customer name
            "channels": [
                {
                    "channel_id": channelId,
                    "pinned_channel_sequence": null,
                    "is_default_for_customer": true
                }
            ]
        })
    }).then(response => {
        if (!response.ok) {
            throw new Error(`Unexpected status code: ${response.status}`);
        }
    });
}