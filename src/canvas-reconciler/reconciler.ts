// eslint-disable-next-line agentverse/no-sideways-capability-imports
import { canvasStore } from '@/canvas-document/store';
import { CaoClient, caoClient } from '@/api/cao-client';
import { CanvasDocument, CanvasNode, CanvasEdge } from '@/shared/canvas-types';
import { useDeployStore, DeployStep } from './deploy-store';
import { resolveSessionEnv } from '@/api/session-discovery';
import { useSessionStore } from '@/api/session-store';

export class EntryPointChangedError extends Error {
  constructor() {
    super('Changing the entry point requires Tear Down');
    this.name = 'EntryPointChangedError';
  }
}

// Helper to compare string arrays
function arraysEqual(a: string[], b: string[]): boolean {
  if (a.length !== b.length) return false;
  const sortedA = [...a].sort();
  const sortedB = [...b].sort();
  return sortedA.every((val, index) => val === sortedB[index]);
}

// 9.1 Profile-markdown generator: produces YAML frontmatter + system prompt body
export function generateProfileMarkdown(data: {
  name: string;
  role: string;
  provider: string;
  model?: string;
  allowedTools: string[];
  systemPrompt: string;
  session_id?: string;
}): string {
  const lines: string[] = ['---'];
  lines.push(`name: ${data.name}`);
  lines.push(`role: ${data.role}`);
  lines.push(`provider: ${data.provider}`);
  if (data.model) {
    lines.push(`model: ${data.model}`);
  }
  if (data.session_id) {
    lines.push(`session_id: ${data.session_id}`);
  }
  if (data.allowedTools && data.allowedTools.length > 0) {
    lines.push('allowedTools:');
    for (const tool of data.allowedTools) {
      lines.push(`  - ${tool}`);
    }
  } else {
    lines.push('allowedTools: []');
  }
  lines.push('---');
  lines.push(data.systemPrompt);
  return lines.join('\n');
}

// 9.2 Supervisor-prompt augmentation: appends the "canvas topology" block
export function augmentSupervisorPrompt(node: CanvasNode, canvas: CanvasDocument): string {
  const outgoingEdges = canvas.edges.filter((edge) => edge.source === node.id);

  const handoffTargets: string[] = [];
  const assignTargets: string[] = [];
  const sendMessageTargets: string[] = [];

  for (const edge of outgoingEdges) {
    const targetNode = canvas.nodes.find((n) => n.id === edge.target);
    if (!targetNode) continue;
    const targetProfileName = `${targetNode.data.profile_name}_${targetNode.id.replace(/-/g, '_')}`;
    const desc = `- ${targetNode.data.display_name} (Profile: ${targetProfileName})`;

    if (edge.type === 'handoff') {
      handoffTargets.push(desc);
    } else if (edge.type === 'assign') {
      assignTargets.push(desc);
    } else if (edge.type === 'send_message') {
      sendMessageTargets.push(desc);
    }
  }

  let block = '\n\n### Canvas Topology\n';
  block += 'Allowed Handoff Targets:\n' + (handoffTargets.length > 0 ? handoffTargets.join('\n') : 'None') + '\n\n';
  block += 'Allowed Assign Targets:\n' + (assignTargets.length > 0 ? assignTargets.join('\n') : 'None') + '\n\n';
  block += 'Allowed Send Message Targets:\n' + (sendMessageTargets.length > 0 ? sendMessageTargets.join('\n') : 'None');

  return node.data.system_prompt + block;
}

export function didEdgesChange(oldEdges: CanvasEdge[], newEdges: CanvasEdge[]): boolean {
  if (oldEdges.length !== newEdges.length) return true;
  for (const newEdge of newEdges) {
    const oldEdge = oldEdges.find((e) => e.id === newEdge.id);
    if (!oldEdge) return true;
    if (oldEdge.source !== newEdge.source || oldEdge.target !== newEdge.target || oldEdge.type !== newEdge.type) {
      return true;
    }
  }
  return false;
}

// 9.3 & 9.4 & 9.9 Reconciler driver, state transitions, and diff-based edit-after-deploy
export async function reconcileCanvas(
  canvasId: string,
  editedCanvas?: CanvasDocument,
  client: CaoClient = caoClient
): Promise<CanvasDocument> {
  // Retrieve the existing canvas in database
  const existingCanvas = await canvasStore.get(canvasId);
  if (!existingCanvas) throw new Error('Canvas not found');

  const targetCanvas = editedCanvas || existingCanvas;

  // Pre-flight validation
  const entryPointNode = targetCanvas.nodes.find((n) => n.data.is_entry_point);
  if (!entryPointNode) {
    throw new Error('Canvas must have an entry point');
  }
  const multipleEntryPoints = targetCanvas.nodes.filter((n) => n.data.is_entry_point).length > 1;
  if (multipleEntryPoints) {
    throw new Error('Canvas cannot have multiple entry points');
  }

  const isFirstDeploy = existingCanvas.deploy_state.status === 'draft';

  // If edited canvas is provided, check if entry point changed
  if (editedCanvas && !isFirstDeploy) {
    const oldEntryPoint = existingCanvas.nodes.find((n) => n.data.is_entry_point);
    const newEntryPoint = editedCanvas.nodes.find((n) => n.data.is_entry_point);
    if (oldEntryPoint && newEntryPoint && oldEntryPoint.id !== newEntryPoint.id) {
      throw new EntryPointChangedError();
    }
  }

  // Compute diffs if editedCanvas is provided and not first deploy
  const nodesToAdd: CanvasNode[] = [];
  const nodesToRemove: string[] = []; // node IDs
  const nodesToUpdate: CanvasNode[] = [];

  const terminalMap = existingCanvas.deploy_state.terminal_map || {};
  const snapshots = existingCanvas.deploy_state.profile_snapshots || {};

  if (editedCanvas && !isFirstDeploy) {
    const currentNodesMap = new Map(targetCanvas.nodes.map((n) => [n.id, n]));

    // Added nodes
    for (const [id, node] of currentNodesMap.entries()) {
      if (!terminalMap[id]) {
        nodesToAdd.push(node);
      }
    }

    // Removed nodes
    for (const id of Object.keys(terminalMap)) {
      if (!currentNodesMap.has(id)) {
        nodesToRemove.push(id);
      }
    }

    // Updated nodes
    for (const [id, node] of currentNodesMap.entries()) {
      if (terminalMap[id] && snapshots[id]) {
        const snapshot = snapshots[id];
        const providerDefault = targetCanvas.config.provider_default;
        const systemPromptChanged = node.data.system_prompt !== snapshot.system_prompt;
        const toolsChanged = !arraysEqual(node.data.allowedTools || [], snapshot.allowedTools || []);
        const modelChanged = (node.data.model || '') !== (snapshot.model || '');
        const providerChanged = (node.data.provider || providerDefault) !== (snapshot.provider || '');
        const sessionChanged = (node.data.session_id || '') !== (snapshot.session_id || '');

        if (systemPromptChanged || toolsChanged || modelChanged || providerChanged || sessionChanged) {
          nodesToUpdate.push(node);
        }
      }
    }

    const edgesChanged = didEdgesChange(existingCanvas.edges, targetCanvas.edges);

    // If no CAO action is needed, save edited canvas and return
    if (nodesToAdd.length === 0 && nodesToRemove.length === 0 && nodesToUpdate.length === 0) {
      targetCanvas.deploy_state.edge_change_advisory = edgesChanged;
      // Preserve terminal map, snapshots, and status
      targetCanvas.deploy_state.status = existingCanvas.deploy_state.status;
      targetCanvas.deploy_state.terminal_map = terminalMap;
      targetCanvas.deploy_state.profile_snapshots = snapshots;
      targetCanvas.deploy_state.session_name = existingCanvas.deploy_state.session_name;
      return await canvasStore.save(targetCanvas);
    }
  }

  // Set up session name
  let sessionName =
    targetCanvas.deploy_state.session_name ||
    targetCanvas.config.session_name ||
    `session_${targetCanvas.id.replace(/-/g, '_')}`;

  // Start deployment progress in store
  const steps: DeployStep[] = [];
  if (isFirstDeploy || !editedCanvas) {
    // Standard/Retry deploy steps
    steps.push({ id: 'install-profiles', label: 'Install agent profiles', status: 'pending' });
    steps.push({ id: 'create-session', label: 'Create CAO session', status: 'pending' });
    
    // For non-entry-point nodes
    const nonEntryNodes = targetCanvas.nodes.filter((n) => !n.data.is_entry_point);
    nonEntryNodes.forEach((node) => {
      steps.push({
        id: `add-terminal-${node.id}`,
        label: `Add terminal: ${node.data.display_name}`,
        status: terminalMap[node.id] ? 'success' : 'pending',
      });
    });
    steps.push({ id: 'finalize', label: 'Finalize deploy state', status: 'pending' });
  } else {
    // Diff-based deploy steps
    for (const nodeToRemoveId of nodesToRemove) {
      steps.push({
        id: `remove-terminal-${nodeToRemoveId}`,
        label: `Remove terminal: ${nodeToRemoveId.substring(0, 8)}`,
        status: 'pending',
      });
    }
    for (const nodeToUpdate of nodesToUpdate) {
      steps.push({
        id: `update-terminal-${nodeToUpdate.id}`,
        label: `Update terminal: ${nodeToUpdate.data.display_name}`,
        status: 'pending',
      });
    }
    for (const nodeToAdd of nodesToAdd) {
      steps.push({
        id: `add-terminal-${nodeToAdd.id}`,
        label: `Add terminal: ${nodeToAdd.data.display_name}`,
        status: 'pending',
      });
    }
    steps.push({ id: 'finalize', label: 'Finalize deploy state', status: 'pending' });
  }

  useDeployStore.getState().startDeploy(targetCanvas.id, steps);

  // Transition status to deploying
  targetCanvas.deploy_state = {
    ...targetCanvas.deploy_state,
    status: 'deploying',
    session_name: sessionName,
    errors: [],
    terminal_map: { ...terminalMap },
    profile_snapshots: { ...snapshots },
    edge_change_advisory: false,
  };
  let currentCanvasDoc = await canvasStore.save(targetCanvas);

  let anyCallSucceeded = Object.keys(currentCanvasDoc.deploy_state.terminal_map || {}).length > 0;

  try {
    if (isFirstDeploy || !editedCanvas) {
      // 1. Install agent profiles
      useDeployStore.getState().updateStepStatus('install-profiles', 'in_flight');
      
      for (const node of currentCanvasDoc.nodes) {
        if (currentCanvasDoc.deploy_state.terminal_map?.[node.id]) {
          continue; // Already deployed
        }

        const profileName = `${node.data.profile_name}_${node.id.replace(/-/g, '_')}`;
        
        // Augment if supervisor or entry point
        let finalSystemPrompt = node.data.system_prompt;
        if (node.data.is_entry_point || node.data.role === 'supervisor') {
          finalSystemPrompt = augmentSupervisorPrompt(node, currentCanvasDoc);
        }

        const profileMarkdown = generateProfileMarkdown({
          name: profileName,
          role: node.data.role,
          provider: node.data.provider || currentCanvasDoc.config.provider_default,
          model: node.data.model,
          allowedTools: node.data.allowedTools || [],
          systemPrompt: finalSystemPrompt,
          session_id: node.data.session_id,
        });

        // Save before CAO call (atomic persistence)
        await canvasStore.save(currentCanvasDoc);

        try {
          await client.installProfile(profileMarkdown);
          anyCallSucceeded = true;

          currentCanvasDoc.deploy_state.profile_snapshots = {
            ...currentCanvasDoc.deploy_state.profile_snapshots,
            [node.id]: {
              system_prompt: node.data.system_prompt,
              allowedTools: node.data.allowedTools || [],
              model: node.data.model || '',
              provider: node.data.provider || currentCanvasDoc.config.provider_default,
              session_id: node.data.session_id || '',
            },
          };
          await canvasStore.save(currentCanvasDoc);
        } catch (err: unknown) {
          const errorMsg = err instanceof Error ? err.message : String(err);
          currentCanvasDoc.deploy_state.errors = [
            ...(currentCanvasDoc.deploy_state.errors || []),
            { node_id: node.id, error: errorMsg },
          ];
          await canvasStore.save(currentCanvasDoc);
          useDeployStore.getState().updateStepStatus('install-profiles', 'failed');
          throw err;
        }
      }
      useDeployStore.getState().updateStepStatus('install-profiles', 'success');

      // 2. Create session (only if entry point has no terminal ID)
      const entryPointTerminalId = currentCanvasDoc.deploy_state.terminal_map?.[entryPointNode.id];
      if (!entryPointTerminalId) {
        useDeployStore.getState().updateStepStatus('create-session', 'in_flight');
        const entryPointProfileName = `${entryPointNode.data.profile_name}_${entryPointNode.id.replace(/-/g, '_')}`;

        // Save before call
        await canvasStore.save(currentCanvasDoc);

        try {
          let terminalEnv: Record<string, string> = {};
          if (entryPointNode.data.session_id) {
            const sessionStore = useSessionStore.getState();
            const session = sessionStore.getSession(entryPointNode.data.session_id);
            if (session) {
              terminalEnv = resolveSessionEnv(session, entryPointNode.data.model);
            }
          }

          const session = await client.createSession({
            provider: entryPointNode.data.provider || currentCanvasDoc.config.provider_default,
            profile: entryPointProfileName,
            working_directory: currentCanvasDoc.config.working_directory || '~',
            env_vars: Object.keys(terminalEnv).length > 0 ? terminalEnv : undefined,
          });
          anyCallSucceeded = true;
          sessionName = session.name;
          currentCanvasDoc.deploy_state.session_name = session.name;

          // Find terminal ID
          let termId: string | undefined;
          if (session.terminals && session.terminals.length > 0) {
            const matched = session.terminals.find((t) => t.profile === entryPointProfileName);
            if (matched) termId = matched.id;
          }

          if (!termId) {
            const terminalsList = await client.listTerminalsInSession(session.name);
            const matched = terminalsList.find((t) => t.profile === entryPointProfileName);
            if (matched) termId = matched.id;
          }

          if (!termId) {
            throw new Error('Entry point terminal not found in created session');
          }

          currentCanvasDoc.deploy_state.terminal_map = {
            ...currentCanvasDoc.deploy_state.terminal_map,
            [entryPointNode.id]: termId,
          };
          await canvasStore.save(currentCanvasDoc);
          useDeployStore.getState().updateStepStatus('create-session', 'success');
        } catch (err: unknown) {
          const errorMsg = err instanceof Error ? err.message : String(err);
          currentCanvasDoc.deploy_state.errors = [
            ...(currentCanvasDoc.deploy_state.errors || []),
            { node_id: entryPointNode.id, error: errorMsg },
          ];
          await canvasStore.save(currentCanvasDoc);
          useDeployStore.getState().updateStepStatus('create-session', 'failed');
          throw err;
        }
      } else {
        useDeployStore.getState().updateStepStatus('create-session', 'success');
      }

      // 3. Add terminals for non-entry-point nodes
      for (const node of currentCanvasDoc.nodes) {
        if (node.data.is_entry_point) continue;
        const existingTerminalId = currentCanvasDoc.deploy_state.terminal_map?.[node.id];
        if (existingTerminalId) {
          continue; // Already added
        }

        const stepId = `add-terminal-${node.id}`;
        useDeployStore.getState().updateStepStatus(stepId, 'in_flight');

        const profileName = `${node.data.profile_name}_${node.id.replace(/-/g, '_')}`;

        // Save before call
        await canvasStore.save(currentCanvasDoc);

        try {
          let terminalEnv: Record<string, string> = {};
          if (node.data.session_id) {
            const sessionStore = useSessionStore.getState();
            const session = sessionStore.getSession(node.data.session_id);
            if (session) {
              terminalEnv = resolveSessionEnv(session, node.data.model);
            }
          }

          const terminal = await client.addTerminalToSession(sessionName, {
            provider: node.data.provider || currentCanvasDoc.config.provider_default,
            profile: profileName,
            working_directory: currentCanvasDoc.config.working_directory || '~',
            env_vars: Object.keys(terminalEnv).length > 0 ? terminalEnv : undefined,
          });
          anyCallSucceeded = true;

          currentCanvasDoc.deploy_state.terminal_map = {
            ...currentCanvasDoc.deploy_state.terminal_map,
            [node.id]: terminal.id,
          };
          await canvasStore.save(currentCanvasDoc);
          useDeployStore.getState().updateStepStatus(stepId, 'success');
        } catch (err: unknown) {
          const errorMsg = err instanceof Error ? err.message : String(err);
          currentCanvasDoc.deploy_state.errors = [
            ...(currentCanvasDoc.deploy_state.errors || []),
            { node_id: node.id, error: errorMsg },
          ];
          await canvasStore.save(currentCanvasDoc);
          useDeployStore.getState().updateStepStatus(stepId, 'failed');
          throw err;
        }
      }

    } else {
      // Diff-based edit-after-deploy delta application
      // 1. Remove terminals for removed nodes
      for (const nodeToRemoveId of nodesToRemove) {
        const stepId = `remove-terminal-${nodeToRemoveId}`;
        useDeployStore.getState().updateStepStatus(stepId, 'in_flight');
        const terminalId = currentCanvasDoc.deploy_state.terminal_map?.[nodeToRemoveId];

        if (terminalId) {
          // Save before call
          await canvasStore.save(currentCanvasDoc);

          try {
            await client.deleteTerminal(terminalId);
            anyCallSucceeded = true;

            const nextTerminalMap = { ...currentCanvasDoc.deploy_state.terminal_map };
            delete nextTerminalMap[nodeToRemoveId];
            currentCanvasDoc.deploy_state.terminal_map = nextTerminalMap;

            const nextSnapshots = { ...currentCanvasDoc.deploy_state.profile_snapshots };
            delete nextSnapshots[nodeToRemoveId];
            currentCanvasDoc.deploy_state.profile_snapshots = nextSnapshots;

            await canvasStore.save(currentCanvasDoc);
            useDeployStore.getState().updateStepStatus(stepId, 'success');
          } catch (err: unknown) {
            const errorMsg = err instanceof Error ? err.message : String(err);
            currentCanvasDoc.deploy_state.errors = [
              ...(currentCanvasDoc.deploy_state.errors || []),
              { node_id: nodeToRemoveId, error: errorMsg },
            ];
            await canvasStore.save(currentCanvasDoc);
            useDeployStore.getState().updateStepStatus(stepId, 'failed');
            throw err;
          }
        } else {
          useDeployStore.getState().updateStepStatus(stepId, 'success');
        }
      }

      // 2. Update terminals for changed nodes (kill, install profile, add terminal)
      for (const nodeToUpdate of nodesToUpdate) {
        const stepId = `update-terminal-${nodeToUpdate.id}`;
        useDeployStore.getState().updateStepStatus(stepId, 'in_flight');
        const oldTerminalId = currentCanvasDoc.deploy_state.terminal_map?.[nodeToUpdate.id];

        // Save before profile install + kill terminal
        await canvasStore.save(currentCanvasDoc);

        try {
          const profileName = `${nodeToUpdate.data.profile_name}_${nodeToUpdate.id.replace(/-/g, '_')}`;
          
          let finalSystemPrompt = nodeToUpdate.data.system_prompt;
          if (nodeToUpdate.data.is_entry_point || nodeToUpdate.data.role === 'supervisor') {
            finalSystemPrompt = augmentSupervisorPrompt(nodeToUpdate, currentCanvasDoc);
          }

          const profileMarkdown = generateProfileMarkdown({
            name: profileName,
            role: nodeToUpdate.data.role,
            provider: nodeToUpdate.data.provider || currentCanvasDoc.config.provider_default,
            model: nodeToUpdate.data.model,
            allowedTools: nodeToUpdate.data.allowedTools || [],
            systemPrompt: finalSystemPrompt,
            session_id: nodeToUpdate.data.session_id,
          });

          // Install updated profile
          await client.installProfile(profileMarkdown);
          anyCallSucceeded = true;

          // Delete old terminal
          if (oldTerminalId) {
            try {
              await client.deleteTerminal(oldTerminalId);
            } catch (deleteErr) {
              console.warn('Failed to delete old terminal during update, proceeding with create:', deleteErr);
            }
          }

          // Save state showing old terminal deleted
          const nextTerminalMap = { ...currentCanvasDoc.deploy_state.terminal_map };
          delete nextTerminalMap[nodeToUpdate.id];
          currentCanvasDoc.deploy_state.terminal_map = nextTerminalMap;
          await canvasStore.save(currentCanvasDoc);

          // Add new terminal
          let terminalEnv: Record<string, string> = {};
          if (nodeToUpdate.data.session_id) {
            const sessionStore = useSessionStore.getState();
            const session = sessionStore.getSession(nodeToUpdate.data.session_id);
            if (session) {
              terminalEnv = resolveSessionEnv(session, nodeToUpdate.data.model);
            }
          }

          const terminal = await client.addTerminalToSession(sessionName, {
            provider: nodeToUpdate.data.provider || currentCanvasDoc.config.provider_default,
            profile: profileName,
            working_directory: currentCanvasDoc.config.working_directory || '~',
            env_vars: Object.keys(terminalEnv).length > 0 ? terminalEnv : undefined,
          });

          currentCanvasDoc.deploy_state.terminal_map = {
            ...currentCanvasDoc.deploy_state.terminal_map,
            [nodeToUpdate.id]: terminal.id,
          };
          currentCanvasDoc.deploy_state.profile_snapshots = {
            ...currentCanvasDoc.deploy_state.profile_snapshots,
            [nodeToUpdate.id]: {
              system_prompt: nodeToUpdate.data.system_prompt,
              allowedTools: nodeToUpdate.data.allowedTools || [],
              model: nodeToUpdate.data.model || '',
              provider: nodeToUpdate.data.provider || currentCanvasDoc.config.provider_default,
              session_id: nodeToUpdate.data.session_id || '',
            },
          };
          await canvasStore.save(currentCanvasDoc);
          useDeployStore.getState().updateStepStatus(stepId, 'success');
        } catch (err: unknown) {
          const errorMsg = err instanceof Error ? err.message : String(err);
          currentCanvasDoc.deploy_state.errors = [
            ...(currentCanvasDoc.deploy_state.errors || []),
            { node_id: nodeToUpdate.id, error: errorMsg },
          ];
          await canvasStore.save(currentCanvasDoc);
          useDeployStore.getState().updateStepStatus(stepId, 'failed');
          throw err;
        }
      }

      // 3. Add terminals for new nodes
      for (const nodeToAdd of nodesToAdd) {
        const stepId = `add-terminal-${nodeToAdd.id}`;
        useDeployStore.getState().updateStepStatus(stepId, 'in_flight');

        // Save before profile install
        await canvasStore.save(currentCanvasDoc);

        try {
          const profileName = `${nodeToAdd.data.profile_name}_${nodeToAdd.id.replace(/-/g, '_')}`;
          
          let finalSystemPrompt = nodeToAdd.data.system_prompt;
          if (nodeToAdd.data.is_entry_point || nodeToAdd.data.role === 'supervisor') {
            finalSystemPrompt = augmentSupervisorPrompt(nodeToAdd, currentCanvasDoc);
          }

          const profileMarkdown = generateProfileMarkdown({
            name: profileName,
            role: nodeToAdd.data.role,
            provider: nodeToAdd.data.provider || currentCanvasDoc.config.provider_default,
            model: nodeToAdd.data.model,
            allowedTools: nodeToAdd.data.allowedTools || [],
            systemPrompt: finalSystemPrompt,
            session_id: nodeToAdd.data.session_id,
          });

          await client.installProfile(profileMarkdown);
          anyCallSucceeded = true;

          let terminalEnv: Record<string, string> = {};
          if (nodeToAdd.data.session_id) {
            const sessionStore = useSessionStore.getState();
            const session = sessionStore.getSession(nodeToAdd.data.session_id);
            if (session) {
              terminalEnv = resolveSessionEnv(session, nodeToAdd.data.model);
            }
          }

          const terminal = await client.addTerminalToSession(sessionName, {
            provider: nodeToAdd.data.provider || currentCanvasDoc.config.provider_default,
            profile: profileName,
            working_directory: currentCanvasDoc.config.working_directory || '~',
            env_vars: Object.keys(terminalEnv).length > 0 ? terminalEnv : undefined,
          });

          currentCanvasDoc.deploy_state.terminal_map = {
            ...currentCanvasDoc.deploy_state.terminal_map,
            [nodeToAdd.id]: terminal.id,
          };
          currentCanvasDoc.deploy_state.profile_snapshots = {
            ...currentCanvasDoc.deploy_state.profile_snapshots,
            [nodeToAdd.id]: {
              system_prompt: nodeToAdd.data.system_prompt,
              allowedTools: nodeToAdd.data.allowedTools || [],
              model: nodeToAdd.data.model || '',
              provider: nodeToAdd.data.provider || currentCanvasDoc.config.provider_default,
              session_id: nodeToAdd.data.session_id || '',
            },
          };
          await canvasStore.save(currentCanvasDoc);
          useDeployStore.getState().updateStepStatus(stepId, 'success');
        } catch (err: unknown) {
          const errorMsg = err instanceof Error ? err.message : String(err);
          currentCanvasDoc.deploy_state.errors = [
            ...(currentCanvasDoc.deploy_state.errors || []),
            { node_id: nodeToAdd.id, error: errorMsg },
          ];
          await canvasStore.save(currentCanvasDoc);
          useDeployStore.getState().updateStepStatus(stepId, 'failed');
          throw err;
        }
      }
    }

    // Finalize deployment
    useDeployStore.getState().updateStepStatus('finalize', 'in_flight');
    currentCanvasDoc.deploy_state.status = 'deployed';
    currentCanvasDoc.deploy_state.last_deployed = new Date().toISOString();
    currentCanvasDoc.deploy_state.errors = [];
    
    // Save final canvas
    currentCanvasDoc = await canvasStore.save(currentCanvasDoc);
    useDeployStore.getState().updateStepStatus('finalize', 'success');

  } catch (err) {
    if (anyCallSucceeded) {
      currentCanvasDoc.deploy_state.status = 'degraded';
    } else {
      // rollback to draft
      currentCanvasDoc.deploy_state.status = 'draft';
      currentCanvasDoc.deploy_state.terminal_map = {};
      currentCanvasDoc.deploy_state.session_name = undefined;
      currentCanvasDoc.deploy_state.profile_snapshots = {};
    }
    await canvasStore.save(currentCanvasDoc);
    throw err;
  }

  return currentCanvasDoc;
}

// 9.7 Tear Down
export async function tearDownCanvas(
  canvasId: string,
  client: CaoClient = caoClient
): Promise<CanvasDocument> {
  const canvas = await canvasStore.get(canvasId);
  if (!canvas) throw new Error('Canvas not found');

  const sessionName = canvas.deploy_state.session_name;
  if (!sessionName) {
    canvas.deploy_state = {
      status: 'draft',
      terminal_map: {},
      profile_snapshots: {},
    };
    return await canvasStore.save(canvas);
  }

  await client.deleteSession(sessionName);
  canvas.deploy_state = {
    status: 'draft',
    terminal_map: {},
    profile_snapshots: {},
  };
  return await canvasStore.save(canvas);
}

// Helper to cancel ongoing deployment
export async function cancelDeploy(canvasId: string): Promise<CanvasDocument> {
  const canvas = await canvasStore.get(canvasId);
  if (!canvas) throw new Error('Canvas not found');

  const hasTerminals = Object.keys(canvas.deploy_state.terminal_map || {}).length > 0;
  canvas.deploy_state.status = hasTerminals ? 'degraded' : 'draft';
  useDeployStore.getState().clearDeploy();
  return await canvasStore.save(canvas);
}
