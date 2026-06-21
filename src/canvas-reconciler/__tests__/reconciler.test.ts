import { describe, expect, it, beforeEach, vi } from 'vitest';
import { reconcileCanvas, tearDownCanvas, cancelDeploy, EntryPointChangedError } from '../reconciler';
// eslint-disable-next-line agentverse/no-sideways-capability-imports
import { canvasStore } from '@/canvas-document/store';
import { CanvasDocument, CanvasNode, CanvasEdge } from '@/shared/canvas-types';
import { useDeployStore } from '../deploy-store';

describe('Canvas Reconciler', () => {
  let doc: CanvasDocument;
  let mockGoCoreClient: any;

  beforeEach(async () => {
    // Reset Zustand store
    useDeployStore.getState().clearDeploy();

    // Create a 3-node canvas draft
    doc = canvasStore.createDraft();
    doc.name = 'Test Reconciler Canvas';
    doc.config.provider_default = 'openai';

    const supervisor: CanvasNode = {
      id: '00000000-0000-4000-8000-000000000001',
      type: 'agent',
      position: { x: 100, y: 100 },
      data: {
        profile_name: 'supervisor',
        display_name: 'Supervisor Node',
        role: 'supervisor',
        provider: 'openai',
        model: 'gpt-4',
        system_prompt: 'Coordinates team.',
        allowedTools: ['handoff'],
        is_entry_point: true,
      },
    };

    const developer: CanvasNode = {
      id: '00000000-0000-4000-8000-000000000002',
      type: 'agent',
      position: { x: 100, y: 300 },
      data: {
        profile_name: 'developer',
        display_name: 'Developer Node',
        role: 'developer',
        provider: 'anthropic',
        model: 'claude-3',
        system_prompt: 'Writes code.',
        allowedTools: ['shell'],
        is_entry_point: false,
      },
    };

    const reviewer: CanvasNode = {
      id: '00000000-0000-4000-8000-000000000003',
      type: 'agent',
      position: { x: 100, y: 500 },
      data: {
        profile_name: 'reviewer',
        display_name: 'Reviewer Node',
        role: 'reviewer',
        provider: 'openai',
        model: 'gpt-4',
        system_prompt: 'Reviews code.',
        allowedTools: ['read_file'],
        is_entry_point: false,
      },
    };

    const edge1: CanvasEdge = {
      id: 'edge-1',
      source: supervisor.id,
      target: developer.id,
      type: 'handoff',
    };

    const edge2: CanvasEdge = {
      id: 'edge-2',
      source: developer.id,
      target: reviewer.id,
      type: 'handoff',
    };

    doc.nodes = [supervisor, developer, reviewer];
    doc.edges = [edge1, edge2];

    await canvasStore.save(doc);

    // Create a mock GoCoreClient
    mockGoCoreClient = {
      installProfile: vi.fn().mockResolvedValue({ name: 'profile-mock' }),
      createSession: vi.fn().mockResolvedValue({
        name: 'session_test',
        profile: 'supervisor_00000000_0000_4000_8000_000000000001',
        working_directory: '~',
        status: 'active',
        terminals: [
          {
            id: 'term-1',
            profile: 'supervisor_00000000_0000_4000_8000_000000000001',
            status: 'idle',
            working_directory: '~',
          },
        ],
      }),
      addTerminalToSession: vi.fn().mockImplementation((_, input) => {
        const id = input.profile.includes('developer') ? 'term-2' : 'term-3';
        return Promise.resolve({
          id,
          profile: input.profile,
          status: 'idle',
          working_directory: '~',
        });
      }),
      listTerminalsInSession: vi.fn().mockResolvedValue([]),
      deleteSession: vi.fn().mockResolvedValue(undefined),
      deleteTerminal: vi.fn().mockResolvedValue(undefined),
    };
  });

  it('handles full happy-path 3-node deploy', async () => {
    const result = await reconcileCanvas(doc.id, undefined, mockGoCoreClient as any);

    expect(result.deploy_state.status).toBe('deployed');
    expect(result.deploy_state.session_name).toContain('session_');
    expect(result.deploy_state.errors).toEqual([]);
    expect(result.deploy_state.terminal_map).toEqual({
      '00000000-0000-4000-8000-000000000001': 'term-1',
      '00000000-0000-4000-8000-000000000002': 'term-2',
      '00000000-0000-4000-8000-000000000003': 'term-3',
    });

    // Verify snapshots were taken
    expect(result.deploy_state.profile_snapshots?.[doc.nodes[0]!.id]).toEqual({
      system_prompt: 'Coordinates team.',
      allowedTools: ['handoff'],
      model: 'gpt-4',
      provider: 'openai',
    });

    expect(mockGoCoreClient.installProfile).toHaveBeenCalledTimes(3);
    expect(mockGoCoreClient.createSession).toHaveBeenCalledTimes(1);
    expect(mockGoCoreClient.addTerminalToSession).toHaveBeenCalledTimes(2);
  });

  it('handles partial-failure -> degraded state transition', async () => {
    // Make second terminal creation fail
    mockGoCoreClient.addTerminalToSession = vi
      .fn()
      .mockResolvedValueOnce({ id: 'term-2', profile: 'developer', status: 'idle', working_directory: '~' })
      .mockRejectedValueOnce(new Error('GO Core error creating reviewer terminal'));

    await expect(reconcileCanvas(doc.id, undefined, mockGoCoreClient as any)).rejects.toThrow();

    const fetched = await canvasStore.get(doc.id);
    expect(fetched?.deploy_state.status).toBe('degraded');
    expect(fetched?.deploy_state.terminal_map).toEqual({
      '00000000-0000-4000-8000-000000000001': 'term-1',
      '00000000-0000-4000-8000-000000000002': 'term-2',
    });
    expect(fetched?.deploy_state.errors).toHaveLength(1);
    expect(fetched?.deploy_state.errors?.[0]?.node_id).toBe('00000000-0000-4000-8000-000000000003');
    expect(fetched?.deploy_state.errors?.[0]?.error).toBe('GO Core error creating reviewer terminal');
  });

  it('handles all-fail -> draft rollback state transition', async () => {
    // Make first profile install fail
    mockGoCoreClient.installProfile = vi.fn().mockRejectedValue(new Error('Connection failure'));

    await expect(reconcileCanvas(doc.id, undefined, mockGoCoreClient as any)).rejects.toThrow();

    const fetched = await canvasStore.get(doc.id);
    expect(fetched?.deploy_state.status).toBe('draft');
    expect(fetched?.deploy_state.terminal_map).toEqual({});
    expect(fetched?.deploy_state.session_name).toBeUndefined();
  });

  it('handles retry-from-degraded', async () => {
    // Setup initial degraded state in DB
    const degradedDoc = {
      ...doc,
      deploy_state: {
        status: 'degraded' as const,
        session_name: 'session_test',
        terminal_map: {
          '00000000-0000-4000-8000-000000000001': 'term-1',
          '00000000-0000-4000-8000-000000000002': 'term-2',
        },
        profile_snapshots: {
          '00000000-0000-4000-8000-000000000001': {
            system_prompt: 'Coordinates team.',
            allowedTools: ['handoff'],
            model: 'gpt-4',
            provider: 'openai',
          },
          '00000000-0000-4000-8000-000000000002': {
            system_prompt: 'Writes code.',
            allowedTools: ['shell'],
            model: 'claude-3',
            provider: 'anthropic',
          },
        },
        errors: [{ node_id: '00000000-0000-4000-8000-000000000003', error: 'Prior error' }],
      },
    };
    await canvasStore.save(degradedDoc);

    mockGoCoreClient.addTerminalToSession = vi.fn().mockResolvedValue({
      id: 'term-3-retry',
      profile: 'reviewer_00000000_0000_4000_8000_000000000003',
      status: 'idle',
      working_directory: '~',
    });

    const result = await reconcileCanvas(doc.id, undefined, mockGoCoreClient as any);

    expect(result.deploy_state.status).toBe('deployed');
    expect(result.deploy_state.terminal_map).toEqual({
      '00000000-0000-4000-8000-000000000001': 'term-1',
      '00000000-0000-4000-8000-000000000002': 'term-2',
      '00000000-0000-4000-8000-000000000003': 'term-3-retry',
    });
    expect(result.deploy_state.errors).toEqual([]);

    // Profile install should only run for reviewer (not already in snapshots/terminal map)
    expect(mockGoCoreClient.installProfile).toHaveBeenCalledTimes(1);
    expect(mockGoCoreClient.createSession).not.toHaveBeenCalled();
    expect(mockGoCoreClient.addTerminalToSession).toHaveBeenCalledTimes(1);
  });

  it('handles resume and cancel options on reload-mid-deploy', async () => {
    // Setup deploying state in DB (as on reload)
    const deployingDoc = {
      ...doc,
      deploy_state: {
        status: 'deploying' as const,
        session_name: 'session_test',
        terminal_map: {
          '00000000-0000-4000-8000-000000000001': 'term-1',
        },
      },
    };
    await canvasStore.save(deployingDoc);

    // 1. Check Cancel action
    const afterCancel = await cancelDeploy(doc.id);
    expect(afterCancel.deploy_state.status).toBe('degraded');

    // Reset to deploying to test Resume
    deployingDoc.deploy_state.status = 'deploying';
    await canvasStore.save(deployingDoc);

    // 2. Check Resume action
    const afterResume = await reconcileCanvas(doc.id, undefined, mockGoCoreClient as any);
    expect(afterResume.deploy_state.status).toBe('deployed');
    expect(afterResume.deploy_state.terminal_map).toEqual({
      '00000000-0000-4000-8000-000000000001': 'term-1',
      '00000000-0000-4000-8000-000000000002': 'term-2',
      '00000000-0000-4000-8000-000000000003': 'term-3',
    });
  });

  it('handles diff-add-node in edit-after-deploy', async () => {
    // Setup canonical deployed state in DB
    const deployedDoc = {
      ...doc,
      deploy_state: {
        status: 'deployed' as const,
        session_name: 'session_test',
        terminal_map: {
          '00000000-0000-4000-8000-000000000001': 'term-1',
          '00000000-0000-4000-8000-000000000002': 'term-2',
          '00000000-0000-4000-8000-000000000003': 'term-3',
        },
        profile_snapshots: {
          '00000000-0000-4000-8000-000000000001': {
            system_prompt: 'Coordinates team.',
            allowedTools: ['handoff'],
            model: 'gpt-4',
            provider: 'openai',
          },
          '00000000-0000-4000-8000-000000000002': {
            system_prompt: 'Writes code.',
            allowedTools: ['shell'],
            model: 'claude-3',
            provider: 'anthropic',
          },
          '00000000-0000-4000-8000-000000000003': {
            system_prompt: 'Reviews code.',
            allowedTools: ['read_file'],
            model: 'gpt-4',
            provider: 'openai',
          },
        },
      },
    };
    await canvasStore.save(deployedDoc);

    // Edit canvas: add a 4th node (another developer)
    const newDeveloper: CanvasNode = {
      id: '00000000-0000-4000-8000-000000000004',
      type: 'agent',
      position: { x: 100, y: 700 },
      data: {
        profile_name: 'new_developer',
        display_name: 'New Developer',
        role: 'developer',
        provider: 'openai',
        model: 'gpt-4',
        system_prompt: 'Helper.',
        allowedTools: [],
        is_entry_point: false,
      },
    };

    const editedDoc = {
      ...deployedDoc,
      nodes: [...deployedDoc.nodes, newDeveloper],
    };

    mockGoCoreClient.addTerminalToSession = vi.fn().mockResolvedValue({
      id: 'term-4',
      profile: 'new_developer_00000000_0000_4000_8000_000000000004',
      status: 'idle',
      working_directory: '~',
    });

    const result = await reconcileCanvas(doc.id, editedDoc, mockGoCoreClient as any);

    expect(result.deploy_state.status).toBe('deployed');
    expect(result.deploy_state.terminal_map).toEqual({
      '00000000-0000-4000-8000-000000000001': 'term-1',
      '00000000-0000-4000-8000-000000000002': 'term-2',
      '00000000-0000-4000-8000-000000000003': 'term-3',
      '00000000-0000-4000-8000-000000000004': 'term-4',
    });

    // Reconciler should only install 1 profile and add 1 terminal
    expect(mockGoCoreClient.installProfile).toHaveBeenCalledTimes(1);
    expect(mockGoCoreClient.addTerminalToSession).toHaveBeenCalledTimes(1);
    expect(mockGoCoreClient.deleteTerminal).not.toHaveBeenCalled();
  });

  it('handles diff-remove-node in edit-after-deploy', async () => {
    // Setup deployed state
    const deployedDoc = {
      ...doc,
      deploy_state: {
        status: 'deployed' as const,
        session_name: 'session_test',
        terminal_map: {
          '00000000-0000-4000-8000-000000000001': 'term-1',
          '00000000-0000-4000-8000-000000000002': 'term-2',
          '00000000-0000-4000-8000-000000000003': 'term-3',
        },
        profile_snapshots: {
          '00000000-0000-4000-8000-000000000001': {
            system_prompt: 'Coordinates team.',
            allowedTools: ['handoff'],
            model: 'gpt-4',
            provider: 'openai',
          },
          '00000000-0000-4000-8000-000000000002': {
            system_prompt: 'Writes code.',
            allowedTools: ['shell'],
            model: 'claude-3',
            provider: 'anthropic',
          },
          '00000000-0000-4000-8000-000000000003': {
            system_prompt: 'Reviews code.',
            allowedTools: ['read_file'],
            model: 'gpt-4',
            provider: 'openai',
          },
        },
      },
    };
    await canvasStore.save(deployedDoc);

    // Edit: Remove reviewer node (node-3) and remove the dangling edge referring to it
    const editedDoc = {
      ...deployedDoc,
      nodes: [deployedDoc.nodes[0]!, deployedDoc.nodes[1]!],
      edges: deployedDoc.edges.filter(e => e.target !== '00000000-0000-4000-8000-000000000003'),
    };

    const result = await reconcileCanvas(doc.id, editedDoc, mockGoCoreClient as any);

    expect(result.deploy_state.status).toBe('deployed');
    expect(result.deploy_state.terminal_map).toEqual({
      '00000000-0000-4000-8000-000000000001': 'term-1',
      '00000000-0000-4000-8000-000000000002': 'term-2',
    });

    expect(mockGoCoreClient.deleteTerminal).toHaveBeenCalledWith('term-3');
    expect(mockGoCoreClient.installProfile).not.toHaveBeenCalled();
    expect(mockGoCoreClient.addTerminalToSession).not.toHaveBeenCalled();
  });

  it('handles diff-change-profile-content in edit-after-deploy', async () => {
    // Setup deployed state
    const deployedDoc = {
      ...doc,
      deploy_state: {
        status: 'deployed' as const,
        session_name: 'session_test',
        terminal_map: {
          '00000000-0000-4000-8000-000000000001': 'term-1',
          '00000000-0000-4000-8000-000000000002': 'term-2',
        },
        profile_snapshots: {
          '00000000-0000-4000-8000-000000000001': {
            system_prompt: 'Coordinates team.',
            allowedTools: ['handoff'],
            model: 'gpt-4',
            provider: 'openai',
          },
          '00000000-0000-4000-8000-000000000002': {
            system_prompt: 'Writes code.',
            allowedTools: ['shell'],
            model: 'claude-3',
            provider: 'anthropic',
          },
        },
      },
    };
    // Clean up node-3 and its dangling edge from initial doc so we test a 2-node deployment
    deployedDoc.nodes = [deployedDoc.nodes[0]!, deployedDoc.nodes[1]!];
    deployedDoc.edges = deployedDoc.edges.filter(e => e.target !== '00000000-0000-4000-8000-000000000003');
    await canvasStore.save(deployedDoc);

    // Edit: change system prompt for Developer (node-2)
    const developerNode = { ...deployedDoc.nodes[1]! };
    developerNode.data = {
      ...developerNode.data,
      system_prompt: 'Writes clean, tested code.',
    };

    const editedDoc = {
      ...deployedDoc,
      nodes: [deployedDoc.nodes[0]!, developerNode],
    };

    mockGoCoreClient.addTerminalToSession = vi.fn().mockResolvedValue({
      id: 'term-2-new',
      profile: 'developer_00000000_0000_4000_8000_000000000002',
      status: 'idle',
      working_directory: '~',
    });

    const result = await reconcileCanvas(doc.id, editedDoc, mockGoCoreClient as any);

    expect(result.deploy_state.status).toBe('deployed');
    expect(result.deploy_state.terminal_map).toEqual({
      '00000000-0000-4000-8000-000000000001': 'term-1',
      '00000000-0000-4000-8000-000000000002': 'term-2-new',
    });

    expect(mockGoCoreClient.deleteTerminal).toHaveBeenCalledWith('term-2');
    expect(mockGoCoreClient.installProfile).toHaveBeenCalledTimes(1);
    expect(mockGoCoreClient.addTerminalToSession).toHaveBeenCalledTimes(1);
  });

  it('handles diff-display-only edits without triggering GO Core calls', async () => {
    // Setup deployed state
    const deployedDoc = {
      ...doc,
      deploy_state: {
        status: 'deployed' as const,
        session_name: 'session_test',
        terminal_map: {
          '00000000-0000-4000-8000-000000000001': 'term-1',
        },
        profile_snapshots: {
          '00000000-0000-4000-8000-000000000001': {
            system_prompt: 'Coordinates team.',
            allowedTools: ['handoff'],
            model: 'gpt-4',
            provider: 'openai',
          },
        },
      },
    };
    deployedDoc.nodes = [deployedDoc.nodes[0]!];
    deployedDoc.edges = [];
    await canvasStore.save(deployedDoc);

    // Edit: change display name and position (display-only changes)
    const supervisorNode = { ...deployedDoc.nodes[0]! };
    supervisorNode.data = {
      ...supervisorNode.data,
      display_name: 'Renamed Supervisor',
    };
    supervisorNode.position = { x: 500, y: 500 };

    const editedDoc = {
      ...deployedDoc,
      nodes: [supervisorNode],
    };

    const result = await reconcileCanvas(doc.id, editedDoc, mockGoCoreClient as any);

    expect(result.deploy_state.status).toBe('deployed');
    expect(result.nodes[0]!.data.display_name).toBe('Renamed Supervisor');
    expect(result.nodes[0]!.position).toEqual({ x: 500, y: 500 });

    // Ensure absolutely NO GO Core calls were made
    expect(mockGoCoreClient.installProfile).not.toHaveBeenCalled();
    expect(mockGoCoreClient.createSession).not.toHaveBeenCalled();
    expect(mockGoCoreClient.addTerminalToSession).not.toHaveBeenCalled();
    expect(mockGoCoreClient.deleteTerminal).not.toHaveBeenCalled();
  });

  it('blocks entry-point changes by throwing EntryPointChangedError', async () => {
    // Setup deployed state
    const deployedDoc = {
      ...doc,
      deploy_state: {
        status: 'deployed' as const,
        session_name: 'session_test',
        terminal_map: {
          '00000000-0000-4000-8000-000000000001': 'term-1',
          '00000000-0000-4000-8000-000000000002': 'term-2',
        },
      },
    };
    deployedDoc.nodes = [deployedDoc.nodes[0]!, deployedDoc.nodes[1]!];
    deployedDoc.edges = deployedDoc.edges.filter(e => e.target !== '00000000-0000-4000-8000-000000000003');
    await canvasStore.save(deployedDoc);

    // Edit: promote Developer (node-2) to entry point, demote Supervisor (node-1)
    const nodes = [
      {
        ...deployedDoc.nodes[0]!,
        data: { ...deployedDoc.nodes[0]!.data, is_entry_point: false },
      },
      {
        ...deployedDoc.nodes[1]!,
        data: { ...deployedDoc.nodes[1]!.data, is_entry_point: true },
      },
    ];

    const editedDoc = {
      ...deployedDoc,
      nodes,
    };

    await expect(reconcileCanvas(doc.id, editedDoc, mockGoCoreClient as any)).rejects.toThrow(
      EntryPointChangedError
    );

    // Database should be unchanged
    const fetched = await canvasStore.get(doc.id);
    expect(fetched?.deploy_state.status).toBe('deployed');
    expect(fetched?.nodes[0]?.data.is_entry_point).toBe(true);
  });

  it('tears down active sessions and resets state to draft', async () => {
    // Setup deployed state
    const deployedDoc = {
      ...doc,
      deploy_state: {
        status: 'deployed' as const,
        session_name: 'session_test',
        terminal_map: {
          '00000000-0000-4000-8000-000000000001': 'term-1',
        },
        profile_snapshots: {
          '00000000-0000-4000-8000-000000000001': {
            system_prompt: 'Coordinates team.',
            allowedTools: ['handoff'],
            model: 'gpt-4',
            provider: 'openai',
          },
        },
      },
    };
    await canvasStore.save(deployedDoc);

    const result = await tearDownCanvas(doc.id, mockGoCoreClient as any);

    expect(result.deploy_state.status).toBe('draft');
    expect(result.deploy_state.terminal_map).toEqual({});
    expect(result.deploy_state.session_name).toBeUndefined();

    expect(mockGoCoreClient.deleteSession).toHaveBeenCalledWith('session_test');
  });
});
