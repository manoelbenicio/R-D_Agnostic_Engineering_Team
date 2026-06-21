/* eslint-disable agentverse/no-sideways-capability-imports */
import { describe, it, expect, beforeEach, vi } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { FirstRunWizard } from '../FirstRunWizard';
import { goCoreClient } from '@/api';
import { useKeyStore } from '@/api/key-store/store';
import { canvasStore } from '@/canvas-document/store';
import * as idb from '@/shared/storage/idb';

vi.mock('@/shared/storage/idb', () => ({
  dbGet: vi.fn(),
  dbPut: vi.fn().mockResolvedValue(undefined),
}));

describe('FirstRunWizard Component', () => {
  const mockOnClose = vi.fn();

  beforeEach(() => {
    vi.restoreAllMocks();
    mockOnClose.mockClear();
    useKeyStore.setState({
      validated: [],
      statuses: {
        openai: 'unset',
        anthropic: 'unset',
        google: 'unset',
        aws: 'unset',
        azure: 'unset',
        moonshot: 'unset',
        copilot: 'unset',
        opencode: 'unset',
      } as any,
      initialized: true,
    });
  });

  it('renders step 1 and validates GO Core connection successfully', async () => {
    vi.spyOn(goCoreClient, 'getHealth').mockResolvedValue({ status: 'ok' });

    render(
      <MemoryRouter>
        <FirstRunWizard onClose={mockOnClose} />
      </MemoryRouter>
    );

    expect(screen.getByText('Verifying runtime engine connection...')).toBeInTheDocument();

    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'Next: Configure Provider' })).toBeInTheDocument();
    });
  });

  it('allows skipping setup at step 1', async () => {
    vi.spyOn(goCoreClient, 'getHealth').mockResolvedValue({ status: 'ok' });

    render(
      <MemoryRouter>
        <FirstRunWizard onClose={mockOnClose} />
      </MemoryRouter>
    );

    const skipBtn = screen.getByRole('button', { name: 'Skip Setup' });
    fireEvent.click(skipBtn);

    await waitFor(() => {
      expect(idb.dbPut).toHaveBeenCalledWith('app_state', { key: 'wizard_completed', value: true });
      expect(mockOnClose).toHaveBeenCalled();
    });
  });

  it('instantiates template and finishes onboarding at step 3', async () => {
    vi.spyOn(goCoreClient, 'getHealth').mockResolvedValue({ status: 'ok' });
    
    const mockDoc = { id: 'mock-canvas-id', name: 'Code Review Pipeline' };
    vi.spyOn(canvasStore, 'save').mockResolvedValue(mockDoc as any);

    render(
      <MemoryRouter>
        <FirstRunWizard onClose={mockOnClose} />
      </MemoryRouter>
    );

    await waitFor(() => {
      fireEvent.click(screen.getByRole('button', { name: 'Next: Configure Provider' }));
    });

    await waitFor(() => {
      fireEvent.click(screen.getByRole('button', { name: 'Skip & Continue' }));
    });

    expect(screen.getByText('Pick a workspace layout template to start building, or create a completely blank canvas.')).toBeInTheDocument();

    const templateUseButtons = screen.getAllByRole('button', { name: 'Use Template' });
    expect(templateUseButtons.length).toBeGreaterThan(0);
    const firstButton = templateUseButtons[0];
    if (!firstButton) throw new Error('No button found');
    fireEvent.click(firstButton);

    await waitFor(() => {
      expect(canvasStore.save).toHaveBeenCalled();
      expect(idb.dbPut).toHaveBeenCalledWith('app_state', { key: 'wizard_completed', value: true });
      expect(mockOnClose).toHaveBeenCalled();
    });
  });
});
