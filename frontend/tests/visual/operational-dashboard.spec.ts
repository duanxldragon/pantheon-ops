import { expect, test } from '@playwright/test';
import { primeChineseLocale } from '../smoke/helpers/auth';
for (const mode of ['light','dark'] as const) {
 test(`operational dashboard partial error isolation ${mode}`,async ({page},testInfo)=>{
  await primeChineseLocale(page);await page.addInitScript(color=>localStorage.setItem('pantheon_color_mode',color),mode);
  await page.goto('/__visual/operational-workbench?dashboard=1');
  await expect(page.locator('.operational-dashboard-widget')).toHaveCount(4);
  await expect(page.getByText('数据已过期')).toBeVisible();await expect(page.getByText('部分结果')).toBeVisible();
  await expect(page.getByRole('button',{name:'重试'})).toBeVisible();
  await expect(page.locator('.operational-dashboard-widget__items')).toHaveCount(2);
  await page.getByRole('button',{name:'重试'}).focus();await expect(page.getByRole('button',{name:'重试'})).toBeFocused();
  for(const [name,width,height] of [['desktop',1440,900],['mobile',390,844]] as const) {
   await page.setViewportSize({width,height});
   await expect.poll(()=>page.evaluate(()=>document.documentElement.scrollWidth-innerWidth)).toBeLessThanOrEqual(1);
   await page.screenshot({path:testInfo.outputPath(`dashboard-${mode}-${name}.png`),fullPage:true});
  }
  await page.getByRole('button',{name:'重试'}).click();
  await expect(page.getByRole('button',{name:'重试'})).toBeHidden();
  for(const [name,width,height] of [['desktop',1440,900],['mobile',390,844]] as const) {
   await page.setViewportSize({width,height});
   await expect.poll(()=>page.evaluate(()=>document.documentElement.scrollWidth-innerWidth)).toBeLessThanOrEqual(1);
   await page.screenshot({path:testInfo.outputPath(`dashboard-${mode}-${name}-retry-waiting.png`),fullPage:true});
  }

 });
}
