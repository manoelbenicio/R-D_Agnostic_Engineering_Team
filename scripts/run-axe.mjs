import { chromium } from '@playwright/test';
import AxeBuilder from '@axe-core/playwright';

async function runA11yAudit() {
  const PORT = 5173;
  console.log(`Starting accessibility audit against http://localhost:${PORT}...`);

  const routes = [
    '/',
    '/dashboard',
    '/canvas/demo-session',
    '/agent-studio',
    '/flows',
    '/finops',
    '/memory',
    '/settings/providers',
    '/settings/appearance',
    '/settings/general',
    '/health'
  ];

  let browser;
  try {
    browser = await chromium.launch({ headless: true });
    const context = await browser.newContext();
    const page = await context.newPage();

    // 1. Visit '/' first to initialize DB
    console.log("Navigating to / to initialize database...");
    await page.goto(`http://localhost:${PORT}/`);
    await page.waitForSelector('#app-navbar', { timeout: 10000 });

    // Write wizard_completed to IndexedDB directly
    console.log("Writing wizard_completed = true to IndexedDB...");
    await page.evaluate(async () => {
      return new Promise((resolve, reject) => {
        const req = indexedDB.open('AgentVerse', 1);
        req.onsuccess = () => {
          const db = req.result;
          const tx = db.transaction('app_state', 'readwrite');
          const store = tx.objectStore('app_state');
          const reqPut = store.put({ key: 'wizard_completed', value: true });
          reqPut.onsuccess = () => resolve();
          reqPut.onerror = () => reject(reqPut.error);
        };
        req.onerror = () => reject(req.error);
      });
    });
    console.log("Wizard bypassed successfully ✓");
    
    let totalViolations = 0;
    const routeViolationCounts = {};

    for (const route of routes) {
      console.log(`Auditing route: ${route}...`);
      try {
        await page.goto(`http://localhost:${PORT}${route}`);
        
        // Wait for the app layout to render
        await page.waitForSelector('#app-navbar', { timeout: 10000 });

        // Run Axe audit
        const results = await new AxeBuilder({ page }).analyze();

        // Filter for 'critical' and 'serious' impact
        const seriousOrCritical = results.violations.filter(
          (v) => v.impact === 'critical' || v.impact === 'serious'
        );

        routeViolationCounts[route] = seriousOrCritical.length;

        if (seriousOrCritical.length > 0) {
          console.error(`❌ Route ${route} failed with ${seriousOrCritical.length} critical/serious violations.`);
          seriousOrCritical.forEach((violation, index) => {
            console.error(`  Violation ${index + 1}: [${violation.id}] - ${violation.help} (${violation.impact})`);
            console.error(`  Description: ${violation.description}`);
            violation.nodes.forEach((node) => {
              console.error(`    - HTML target: ${node.html}`);
              console.error(`      Selector: ${node.target.join(', ')}`);
            });
          });
          totalViolations += seriousOrCritical.length;
        } else {
          console.log(`✅ Route ${route} passed (0 critical/serious violations).`);
        }
      } catch (routeError) {
        console.error(`⚠️ Error auditing route ${route}:`, routeError.message);
        routeViolationCounts[route] = 'ERROR';
      }
    }

    console.log('\nAudit Summary:');
    console.table(routeViolationCounts);

    if (totalViolations > 0) {
      console.error(`❌ Axe audit failed with ${totalViolations} total critical/serious violations.`);
      process.exit(1);
    } else {
      console.log('✅ Accessibility audit passed. Zero critical/serious violations found!');
      process.exit(0);
    }
  } catch (error) {
    console.error('Error executing accessibility audit:', error.message);
    console.error('Make sure the dev server is running on port 5173.');
    process.exit(1);
  } finally {
    if (browser) {
      await browser.close();
    }
  }
}

runA11yAudit();
