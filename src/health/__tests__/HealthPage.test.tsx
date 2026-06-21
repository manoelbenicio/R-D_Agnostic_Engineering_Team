/* eslint-disable agentverse/no-sideways-capability-imports */
import { describe, it, expect, beforeEach, vi } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { HealthPage } from '../HealthPage';
import { goCoreClient } from '@/api';
import { useKeyStore } from '@/api/key-store/store';

describe('HealthPage Component', () => {
  beforeEach(() => {
    vi.restoreAllMocks();
    vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockImplementation(() => null);
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

  it('renders sections and handles healthy server state', async () => {
    vi.spyOn(goCoreClient, 'getHealth').mockResolvedValue({ status: 'ok' });
    vi.spyOn(goCoreClient, 'listSessions').mockResolvedValue([]);
    vi.spyOn(goCoreClient, 'listProviders').mockResolvedValue([
      { name: 'google', installed: true },
    ]);

    render(
      <MemoryRouter>
        <HealthPage />
      </MemoryRouter>
    );

    expect(screen.getByText(/System Health & Diagnostics/i)).toBeInTheDocument();
    expect(screen.getByText('Server Health')).toBeInTheDocument();
    expect(screen.getByText('Provider Validations (BYOK)')).toBeInTheDocument();
    expect(screen.getByText('Browser Capabilities')).toBeInTheDocument();

    await waitFor(() => {
      expect(screen.getByText('Runtime engine is running and responding.')).toBeInTheDocument();
      expect(screen.getByText('tmux Server is active and operational.')).toBeInTheDocument();
      expect(screen.getByText('Provider: google')).toBeInTheDocument();
    });
  });

  it('handles GO Core server outage and displays fix button', async () => {
    vi.spyOn(goCoreClient, 'getHealth').mockRejectedValue(new Error('Connection timed out'));
    vi.spyOn(goCoreClient, 'listSessions').mockRejectedValue(new Error('Connection timed out'));
    vi.spyOn(goCoreClient, 'listProviders').mockRejectedValue(new Error('Connection timed out'));

    render(
      <MemoryRouter>
        <HealthPage />
      </MemoryRouter>
    );

    await waitFor(() => {
      expect(screen.getByText(/Cannot reach the runtime at http/i)).toBeInTheDocument();
      expect(screen.getAllByRole('button', { name: 'Fix' }).length).toBeGreaterThan(0);
    });
  });

  it('handles microphone verification success', async () => {
    vi.spyOn(goCoreClient, 'getHealth').mockResolvedValue({ status: 'ok' });
    vi.spyOn(goCoreClient, 'listSessions').mockResolvedValue([]);
    vi.spyOn(goCoreClient, 'listProviders').mockResolvedValue([]);

    const mockStream = {
      getTracks: () => [{ stop: vi.fn() }],
    };
    const getUserMediaMock = vi.fn().mockResolvedValue(mockStream);
    
    if (!navigator.mediaDevices) {
      Object.defineProperty(navigator, 'mediaDevices', {
        value: {
          getUserMedia: getUserMediaMock,
        },
        writable: true,
        configurable: true,
      });
    } else {
      Object.defineProperty(navigator.mediaDevices, 'getUserMedia', {
        value: getUserMediaMock,
        writable: true,
        configurable: true,
      });
    }

    render(
      <MemoryRouter>
        <HealthPage />
      </MemoryRouter>
    );

    const testMicButton = screen.getByRole('button', { name: /Test Microphone/i });
    fireEvent.click(testMicButton);

    await waitFor(() => {
      expect(getUserMediaMock).toHaveBeenCalled();
      expect(screen.getByText('Microphone permission granted and active.')).toBeInTheDocument();
    });
  });
});
