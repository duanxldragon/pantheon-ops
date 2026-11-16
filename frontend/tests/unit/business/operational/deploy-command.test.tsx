import { describe, expect, it, vi, afterEach } from 'vitest';
import { apiRequest } from '@duanxldragon/pantheon-base-ui';
import { cancelDeployTask, pauseDeployTask, resumeDeployTask, retryDeployTask, isDeployTaskCommandConflict } from '../../../../src/modules/business/deploy/api';
vi.mock('@duanxldragon/pantheon-base-ui', () => ({ apiRequest: vi.fn() }));
afterEach(() => vi.resetAllMocks());
describe('Viewed task version preconditions', () => {
  for (const command of [cancelDeployTask, pauseDeployTask, resumeDeployTask, retryDeployTask]) {
    it(`${command.name} preserves the viewed version and never replays a conflict`, async () => {
      const conflict = { code: 409, message: 'business.deploy.task.commandConflict' };
      vi.mocked(apiRequest).mockRejectedValue(conflict);
      await expect(command(71, 'viewed-version')).rejects.toBe(conflict);
      expect(apiRequest).toHaveBeenCalledTimes(1);
      expect(apiRequest).toHaveBeenCalledWith(expect.objectContaining({ method: 'post', data: { stateVersion: 'viewed-version' } }));
      expect(isDeployTaskCommandConflict(conflict)).toBe(true);
    });
  }
  it('distinguishes conflict from forbidden and unavailable failures', () => {
    expect(isDeployTaskCommandConflict({ status: 409 })).toBe(true);
    for (const error of [null, new Error('network'), {code:403}, {status:503}]) expect(isDeployTaskCommandConflict(error)).toBe(false);
  });
});
