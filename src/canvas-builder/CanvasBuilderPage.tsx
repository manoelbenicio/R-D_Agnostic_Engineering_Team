/* eslint-disable agentverse/no-sideways-capability-imports */
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import {
  Background,
  Connection,
  Controls,
  Edge,
  EdgeChange,
  EdgeTypes,
  MarkerType,
  MiniMap,
  Node,
  NodeChange,
  NodeTypes,
  ReactFlow,
  ReactFlowProvider,
  applyEdgeChanges,
  applyNodeChanges,
  reconnectEdge,
  useReactFlow,
} from '@xyflow/react';
import '@xyflow/react/dist/style.css';
import { useNavigate, useParams } from 'react-router-dom';
import { Button, Card, Modal } from '@/design-system';
import { NoProvidersNotice } from '@/api/key-store/no-providers-notice';
import { useValidatedProviders } from '@/api/key-store/use-validated-providers';
import { useInstalledCliProviders } from '@/api/use-installed-cli-providers';
import { useKeyStore } from '@/api/key-store/store';
import { canvasStore } from '@/canvas-document/store';
import { CanvasDocument, CanvasEdge, CanvasNode, OrchestrationType } from '@/shared/canvas-types';
import { useToast } from '@/shell/toasts';
import { TemplatePicker } from '@/canvas-templates';
import { VoicePanel } from '@/voice/VoicePanel';
import { useVoiceHotkey } from '@/voice/use-voice-hotkey';
import { useVoiceStore } from '@/voice/store';
import {
  DeployProgressPanel,
  EdgeAdvisoryBanner,
  EntryPointChangedError,
  cancelDeploy,
  reconcileCanvas,
  tearDownCanvas,
} from '@/canvas-reconciler';
import AgentNode from './AgentNode';
import AgentPalette from './AgentPalette';
import BlockConfigurationPanel from './BlockConfigurationPanel';
import { KeyboardShortcutsHelp } from './KeyboardShortcutsHelp';
import { VersionHistory } from './VersionHistory';
import OrchestrationEdge from './OrchestrationEdge';
import { validateCanvasForDeploy } from './deploy-validation';
import { getCanvasProviderOptionsWithCli } from './provider-options';
import { AGENT_COLOR_PALETTE, createAgentNode, StarterRole } from './role-templates';
import './canvas-builder.css';

type FlowNode = Node<Record<string, unknown>, 'agent'>;
type FlowEdge = Edge<Record<string, unknown>, 'orchestration'>;

const nodeTypes: NodeTypes = { agent: AgentNode };
const edgeTypes: EdgeTypes = { orchestration: OrchestrationEdge };

export const CanvasBuilderPage: React.FC = () => (
  <ReactFlowProvider>
    <CanvasBuilderInner />
  </ReactFlowProvider>
);

const CanvasBuilderInner: React.FC = () => {
  const { id } = useParams();
  const navigate = useNavigate();
  const toast = useToast();
  const reactFlow = useReactFlow();
  const validatedProviders = useValidatedProviders();
  const { installed: installedCliProviders } = useInstalledCliProviders();
  const providerOptions = useMemo(
    () => getCanvasProviderOptionsWithCli(validatedProviders, installedCliProviders),
    [validatedProviders, installedCliProviders]
  );
  const setVoiceOpen = useVoiceStore((state) => state.setOpen);
  const keyStoreInit = useKeyStore((state) => state.init);
  const keyStoreInitialized = useKeyStore((state) => state.initialized);

  const [doc, setDoc] = useState<CanvasDocument | null>(null);
  const [selectedNodeId, setSelectedNodeId] = useState<string | null>(null);
  const [isTemplatePickerOpen, setTemplatePickerOpen] = useState(false);
  const [past, setPast] = useState<CanvasDocument[]>([]);
  const [future, setFuture] = useState<CanvasDocument[]>([]);
  const [isTouchOnly, setTouchOnly] = useState(false);
  const [isReconciling, setReconciling] = useState(false);
  const [entryPointDialogOpen, setEntryPointDialogOpen] = useState(false);
  const [contextMenu, setContextMenu] = useState<{ nodeId: string; x: number; y: number } | null>(null);
  const contextMenuRef = useRef<HTMLDivElement>(null);
  const [zenMode, setZenMode] = useState(false);
  const [panelsCollapsed, setPanelsCollapsed] = useState(false);
  const [zoomLevel, setZoomLevel] = useState(1);
  const [shortcutsHelpOpen, setShortcutsHelpOpen] = useState(false);
  const [historyOpen, setHistoryOpen] = useState(false);

  // Autosave bookkeeping: latest doc (for flush on hide/unmount) and the
  // signature of the content last written, to skip redundant persists.
  const docRef = useRef<CanvasDocument | null>(null);
  docRef.current = doc;
  const lastAutosavedRef = useRef<string | null>(null);

  useVoiceHotkey();

  const handleFitView = useCallback(() => {
    void reactFlow.fitView({ padding: 0.16, duration: 300, minZoom: 0.2, maxZoom: 1.4 });
  }, [reactFlow]);

  const handleZoomIn = useCallback(() => {
    void reactFlow.zoomIn();
  }, [reactFlow]);

  const handleZoomOut = useCallback(() => {
    void reactFlow.zoomOut();
  }, [reactFlow]);

  const closeShortcutsHelp = useCallback(() => setShortcutsHelpOpen(false), []);

  useEffect(() => {
    if (!keyStoreInitialized) {
      void keyStoreInit();
    }
  }, [keyStoreInit, keyStoreInitialized]);

  useEffect(() => {
    const media = window.matchMedia('(hover: none) and (pointer: coarse)');
    setTouchOnly(media.matches);
    const listener = (event: MediaQueryListEvent) => setTouchOnly(event.matches);
    media.addEventListener('change', listener);
    return () => media.removeEventListener('change', listener);
  }, []);

  useEffect(() => {
    let cancelled = false;
    async function loadCanvas() {
      if (!id) return;
      const loaded = await canvasStore.get(id);
      if (cancelled) return;
      if (loaded) {
        setDoc(loaded);
        lastAutosavedRef.current = autosaveSignature(loaded);
        setPast([]);
        setFuture([]);
      } else {
        const draft = canvasStore.createDraft();
        setDoc(draft);
        lastAutosavedRef.current = autosaveSignature(draft);
        toast.warning('Canvas not found. Opened a new draft instead.');
      }
    }
    void loadCanvas();
    return () => {
      cancelled = true;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [id]);

  // Autosave: debounce-persist edits to IndexedDB and flush immediately when the
  // tab is hidden/unloaded (e.g. an OAuth popup steals focus) or on unmount, so
  // unsaved canvas work survives reload and navigation.
  useEffect(() => {
    if (!doc) return;
    const status = doc.deploy_state.status;
    if (isReconciling || status === 'deploying') return;

    const signature = autosaveSignature(doc);
    if (signature === lastAutosavedRef.current) return;

    const flush = () => {
      const current = docRef.current;
      if (!current) return;
      const sig = autosaveSignature(current);
      if (sig === lastAutosavedRef.current) return;
      lastAutosavedRef.current = sig;
      void canvasStore.persist(structuredClone(current));
    };

    const timer = window.setTimeout(flush, 800);
    const onHide = () => {
      if (document.visibilityState === 'hidden') flush();
    };
    document.addEventListener('visibilitychange', onHide);
    window.addEventListener('pagehide', flush);
    return () => {
      window.clearTimeout(timer);
      document.removeEventListener('visibilitychange', onHide);
      window.removeEventListener('pagehide', flush);
    };
  }, [doc, isReconciling]);

  const updateDoc = useCallback(
    (updater: (current: CanvasDocument) => CanvasDocument) => {

      setDoc((current) => {
        if (!current) {

          return current;
        }
        if (isReconciling || current.deploy_state.status === 'deploying') {

          return current;
        }
        const nextDoc = updater(structuredClone(current));
        if (
          JSON.stringify(nextDoc.nodes) === JSON.stringify(current.nodes) &&
          JSON.stringify(nextDoc.edges) === JSON.stringify(current.edges) &&
          JSON.stringify(nextDoc.deploy_state) === JSON.stringify(current.deploy_state)
        ) {
          return current;
        }

        setPast((items) => [...items.slice(-19), structuredClone(current)]);
        setFuture([]);
        return nextDoc;
      });
    },
    [isReconciling]
  );

  useEffect(() => {
    const handleAddNodeEvent = (e: Event) => {
      const customEvent = e as CustomEvent<{ role: string; provider?: string }>;
      const { role, provider } = customEvent.detail;
      const options = getCanvasProviderOptionsWithCli(validatedProviders, installedCliProviders);

      const defaultProvider = provider || options[0]?.provider || '';

      
      updateDoc((current) => {
        const newNode = createAgentNode({
          role: role as StarterRole,
          position: { x: 250, y: 150 + current.nodes.length * 50 },
          hasEntryPoint: current.nodes.some((n) => n.data.is_entry_point),
          provider: defaultProvider,
          nodeIndex: current.nodes.length,
        });
        
        const providerModelMap: Record<string, string> = {
          kiro_cli: 'opus-4.8',
          gemini_cli: 'gemini-3.5-flash',
          codex: 'gpt-5.5',
          claude_code: 'claude-sonnet-4-20250514',
        };
        newNode.data.model = providerModelMap[defaultProvider] || '';
        
        const nextNodes = [...current.nodes, newNode];
        return {
          ...current,
          nodes: nextNodes,
        };
      });
      toast.success(`Agent node '${role}' added via voice.`);
    };

    const handleConnectEvent = (e: Event) => {
      const customEvent = e as CustomEvent<{
        source: { type: 'role' | 'name' | 'id'; value: string };
        destination: { type: 'role' | 'name' | 'id'; value: string };
      }>;
      const { source, destination } = customEvent.detail;

      updateDoc((current) => {
        const findNode = (targetInfo: typeof source) => {
          if (targetInfo.type === 'id') {
            return current.nodes.find((n) => n.id === targetInfo.value);
          }
          if (targetInfo.type === 'role') {
            return current.nodes.find((n) => n.data.role === targetInfo.value);
          }
          return current.nodes.find((n) => n.data.display_name.toLowerCase() === targetInfo.value.toLowerCase());
        };

        const srcNode = findNode(source);
        const destNode = findNode(destination);

        if (!srcNode || !destNode) {
          toast.error(`Could not find nodes to connect for voice command.`);
          return current;
        }

        const newEdge: CanvasEdge = {
          id: crypto.randomUUID(),
          source: srcNode.id,
          target: destNode.id,
          type: 'handoff',
          label: 'handoff',
        };

        return {
          ...current,
          edges: [...current.edges, newEdge],
        };
      });
      toast.success(`Connected ${source.value} to ${destination.value} via voice.`);
    };

    window.addEventListener('voice-canvas-add-node', handleAddNodeEvent);
    window.addEventListener('voice-canvas-connect', handleConnectEvent);

    return () => {
      window.removeEventListener('voice-canvas-add-node', handleAddNodeEvent);
      window.removeEventListener('voice-canvas-connect', handleConnectEvent);
    };
  }, [installedCliProviders, updateDoc, validatedProviders, toast]);

  const selectedNode = useMemo(
    () => doc?.nodes.find((node) => node.id === selectedNodeId),
    [doc?.nodes, selectedNodeId]
  );
  const isCanvasLocked = isReconciling || doc?.deploy_state.status === 'deploying';

  const handleEdgeTypeChange = useCallback(
    (edgeId: string, type: OrchestrationType) => {
      updateDoc((current) => ({
        ...current,
        edges: current.edges.map((edge) =>
          edge.id === edgeId ? { ...edge, type, label: type } : edge
        ),
      }));
    },
    [updateDoc]
  );

  const handleEdgeDelete = useCallback(
    (edgeId: string) => {
      updateDoc((current) => ({
        ...current,
        edges: current.edges.filter((edge) => edge.id !== edgeId),
      }));
    },
    [updateDoc]
  );

  const nodes = useMemo<FlowNode[]>(
    () => {
      if (!doc || doc.nodes.length === 0) {
        // Empty canvas: render no nodes. The .canvas-empty-overlay element rendered
        // elsewhere in this page provides the empty-state guidance to the user.
        return [];
      }

      return doc.nodes.map((node) => ({
        id: node.id,
        type: 'agent',
        position: node.position,
        data: node.data as unknown as Record<string, unknown>,
        selected: node.id === selectedNodeId,
        draggable: !isTouchOnly,
        connectable: !isTouchOnly,
      }));
    },
    [doc, isTouchOnly, selectedNodeId]
  );

  const edges = useMemo<FlowEdge[]>(
    () =>
      (doc?.edges ?? []).map((edge) => toFlowEdge(edge, doc?.nodes ?? [], handleEdgeTypeChange, handleEdgeDelete)),
    [doc?.edges, doc?.nodes, handleEdgeTypeChange, handleEdgeDelete]
  );

  const onNodesChange = useCallback(
    (changes: NodeChange<FlowNode>[]) => {
      if (isTouchOnly || isCanvasLocked) return;
      updateDoc((current) => {
        if (current.nodes.length === 0 && !changes.some((c) => c.type === 'add')) {
          return current;
        }
        const currentFlowNodes: FlowNode[] = current.nodes.map((node) => ({
          id: node.id,
          type: 'agent',
          position: node.position,
          data: node.data as unknown as Record<string, unknown>,
          selected: node.id === selectedNodeId,
          draggable: !isTouchOnly,
          connectable: !isTouchOnly,
        }));
        const nextFlowNodes = applyNodeChanges(changes, currentFlowNodes);
        const nextIds = new Set(nextFlowNodes.map((node) => node.id));
        return {
          ...current,
          nodes: current.nodes
            .filter((node) => nextIds.has(node.id))
            .map((node) => {
              const flowNode = nextFlowNodes.find((candidate) => candidate.id === node.id);
              return flowNode
                ? { ...node, position: flowNode.position, data: flowNode.data as unknown as CanvasNode['data'] }
                : node;
            }),
          edges: current.edges.filter((edge) => nextIds.has(edge.source) && nextIds.has(edge.target)),
        };
      });
    },
    [isTouchOnly, isCanvasLocked, selectedNodeId, updateDoc]
  );

  const onEdgesChange = useCallback(
    (changes: EdgeChange<FlowEdge>[]) => {
      if (isTouchOnly || isCanvasLocked) return;
      updateDoc((current) => {
        if (current.edges.length === 0 && !changes.some((c) => c.type === 'add')) {
          return current;
        }
        const currentFlowEdges: FlowEdge[] = current.edges.map((edge) =>
          toFlowEdge(edge, current.nodes, handleEdgeTypeChange, handleEdgeDelete)
        );
        const nextFlowEdges = applyEdgeChanges(changes, currentFlowEdges);
        return {
          ...current,
          edges: nextFlowEdges.map(fromFlowEdge),
        };
      });
    },
    [isTouchOnly, isCanvasLocked, handleEdgeTypeChange, handleEdgeDelete, updateDoc]
  );

  const onConnect = useCallback(
    (connection: Connection) => {
      if (!connection.source || !connection.target || isTouchOnly || isCanvasLocked) return;
      // Prevent self-connections
      if (connection.source === connection.target) return;
      updateDoc((current) => {
        // Prevent duplicate edges between same source→target
        const duplicate = current.edges.some(
          (e) => e.source === connection.source && e.target === connection.target
        );
        if (duplicate) return current;
        return {
          ...current,
          edges: [
            ...current.edges,
            {
              id: crypto.randomUUID(),
              source: connection.source!,
              target: connection.target!,
              type: 'handoff',
              label: 'handoff',
            },
          ],
        };
      });
    },
    [isTouchOnly, isCanvasLocked, updateDoc]
  );

  const edgeReconnectSuccessful = React.useRef(true);

  const onReconnectStart = useCallback(() => {
    edgeReconnectSuccessful.current = false;
  }, []);

  const onReconnect = useCallback(
    (oldEdge: FlowEdge, newConnection: Connection) => {
      edgeReconnectSuccessful.current = true;
      if (!newConnection.source || !newConnection.target || isTouchOnly || isCanvasLocked) return;
      updateDoc((current) => {
        const currentFlowEdges: FlowEdge[] = current.edges.map((edge) =>
          toFlowEdge(edge, current.nodes, handleEdgeTypeChange, handleEdgeDelete)
        );
        const nextFlowEdges = reconnectEdge(oldEdge, newConnection, currentFlowEdges);
        return {
          ...current,
          edges: nextFlowEdges.map(fromFlowEdge),
        };
      });
    },
    [isTouchOnly, isCanvasLocked, handleEdgeTypeChange, handleEdgeDelete, updateDoc]
  );

  const onReconnectEnd = useCallback(
    (_event: MouseEvent | TouchEvent, edge: FlowEdge) => {
      if (!edgeReconnectSuccessful.current) {
        // Dropped in empty space → delete the edge
        updateDoc((current) => ({
          ...current,
          edges: current.edges.filter((e) => e.id !== edge.id),
        }));
      }
      edgeReconnectSuccessful.current = true;
    },
    [updateDoc]
  );

  const updateNode = useCallback(
    (nodeId: string, updater: (node: CanvasNode) => CanvasNode) => {
      updateDoc((current) => {
        const nextNodes = current.nodes.map((node) =>
          node.id === nodeId ? updater(structuredClone(node)) : node
        );
        const promoted = nextNodes.find((node) => node.id === nodeId)?.data.is_entry_point;
        return {
          ...current,
          nodes: promoted
            ? nextNodes.map((node) =>
                node.id === nodeId
                  ? node
                  : { ...node, data: { ...node.data, is_entry_point: false } }
              )
            : nextNodes,
        };
      });
    },
    [updateDoc]
  );

  const handleDrop = useCallback(
    (event: React.DragEvent<HTMLDivElement>) => {
      event.preventDefault();
      if (isTouchOnly) return;
      const role = event.dataTransfer.getData('application/agentverse-role') as StarterRole;
      if (!role) return;
      const position = reactFlow.screenToFlowPosition({ x: event.clientX, y: event.clientY });
      updateDoc((current) => ({
        ...current,
        nodes: [
          ...current.nodes,
          createAgentNode({
            role,
            position,
            hasEntryPoint: current.nodes.some((node) => node.data.is_entry_point),
            nodeIndex: current.nodes.length,
          }),
        ],
      }));
    },
    [isTouchOnly, reactFlow, updateDoc]
  );

  /* ─── Context Menu Handlers ─── */

  const onNodeContextMenu = useCallback(
    (event: React.MouseEvent, node: FlowNode) => {
      event.preventDefault();
      setContextMenu({ nodeId: node.id, x: event.clientX, y: event.clientY });
    },
    []
  );

  const closeContextMenu = useCallback(() => setContextMenu(null), []);

  const handleSetEntryPoint = useCallback(() => {
    if (!contextMenu) return;
    const { nodeId } = contextMenu;
    updateDoc((current) => ({
      ...current,
      nodes: current.nodes.map((node) => ({
        ...node,
        data: {
          ...node.data,
          is_entry_point: node.id === nodeId,
        },
      })),
    }));
    setContextMenu(null);
  }, [contextMenu, updateDoc]);

  const handleDuplicateNode = useCallback(() => {
    if (!contextMenu || !doc) return;
    const { nodeId } = contextMenu;
    const sourceNode = doc.nodes.find((n) => n.id === nodeId);
    if (!sourceNode) { setContextMenu(null); return; }
    const newId = crypto.randomUUID();
    updateDoc((current) => {
      const color = AGENT_COLOR_PALETTE[current.nodes.length % AGENT_COLOR_PALETTE.length];
      const duplicate: CanvasNode = {
        ...structuredClone(sourceNode),
        id: newId,
        position: {
          x: sourceNode.position.x + 48,
          y: sourceNode.position.y + 48,
        },
        data: {
          ...structuredClone(sourceNode.data),
          profile_name: `${sourceNode.data.role}-${newId.slice(0, 8)}`,
          display_name: `${sourceNode.data.display_name} (copy)`,
          is_entry_point: false,
          color,
        },
      };
      return {
        ...current,
        nodes: [...current.nodes, duplicate],
      };
    });
    setContextMenu(null);
  }, [contextMenu, doc, updateDoc]);

  const handleDeleteNode = useCallback(() => {
    if (!contextMenu) return;
    const { nodeId } = contextMenu;
    updateDoc((current) => ({
      ...current,
      nodes: current.nodes.filter((n) => n.id !== nodeId),
      edges: current.edges.filter((e) => e.source !== nodeId && e.target !== nodeId),
    }));
    if (selectedNodeId === contextMenu.nodeId) setSelectedNodeId(null);
    setContextMenu(null);
  }, [contextMenu, selectedNodeId, updateDoc]);

  const handleDeleteAllConnections = useCallback(() => {
    if (!contextMenu) return;
    const { nodeId } = contextMenu;
    updateDoc((current) => ({
      ...current,
      edges: current.edges.filter((e) => e.source !== nodeId && e.target !== nodeId),
    }));
    setContextMenu(null);
  }, [contextMenu, updateDoc]);

  // Click-away listener to close context menu
  useEffect(() => {
    if (!contextMenu) return;
    const handleClickOutside = (event: MouseEvent) => {
      if (contextMenuRef.current && !contextMenuRef.current.contains(event.target as HTMLElement)) {
        setContextMenu(null);
      }
    };
    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, [contextMenu]);

  const handleSave = useCallback(async () => {
    if (!doc) return;
    try {
      setReconciling(true);
      const saved =
        doc.deploy_state.status === 'deployed' || doc.deploy_state.status === 'degraded'
          ? await reconcileCanvas(doc.id, structuredClone(doc))
          : await canvasStore.save(structuredClone(doc));
      setDoc(saved);
      setPast([]);
      setFuture([]);
      toast.success(
        doc.deploy_state.status === 'deployed' || doc.deploy_state.status === 'degraded'
          ? 'Canvas reconciled.'
          : 'Canvas saved.'
      );
    } catch (error) {
      if (error instanceof EntryPointChangedError) {
        setEntryPointDialogOpen(true);
      } else {
        toast.error(error instanceof Error ? error.message : 'Canvas save failed.');
      }
    } finally {
      setReconciling(false);
    }
  }, [doc, toast]);

  const undo = useCallback(() => {
    setPast((items) => {
      const previous = items.at(-1);
      if (!previous || !doc) return items;
      setFuture((futureItems) => [structuredClone(doc), ...futureItems].slice(0, 20));
      setDoc(previous);
      return items.slice(0, -1);
    });
  }, [doc]);

  const redo = useCallback(() => {
    setFuture((items) => {
      const next = items[0];
      if (!next || !doc) return items;
      setPast((pastItems) => [...pastItems.slice(-19), structuredClone(doc)]);
      setDoc(next);
      return items.slice(1);
    });
  }, [doc]);

  useEffect(() => {
    const handler = (event: KeyboardEvent) => {
      const mod = event.ctrlKey || event.metaKey;
      const key = event.key.toLowerCase();

      if (key === '?' && !isEditableTarget(event.target)) {
        event.preventDefault();
        setShortcutsHelpOpen(true);
        return;
      }

      if (mod && event.shiftKey && key === 'f') {
        event.preventDefault();
        setZenMode((prev) => !prev);
        return;
      }

      if (mod && key === '0') {
        event.preventDefault();
        handleFitView();
        return;
      }

      if (mod && (key === '=' || key === '+')) {
        event.preventDefault();
        handleZoomIn();
        return;
      }

      if (mod && key === '-') {
        event.preventDefault();
        handleZoomOut();
        return;
      }

      // Ctrl/Cmd+S → Save
      if (mod && key === 's') {
        event.preventDefault();
        void handleSave();
        return;
      }

      // Ctrl/Cmd+Shift+Z or Ctrl/Cmd+Y → Redo
      if (mod && ((key === 'z' && event.shiftKey) || key === 'y')) {
        event.preventDefault();
        redo();
        return;
      }

      // Ctrl/Cmd+Z → Undo
      if (mod && key === 'z') {
        event.preventDefault();
        undo();
        return;
      }

      // Escape → Deselect (or exit zen mode)
      if (key === 'escape') {
        if (shortcutsHelpOpen) {
          setShortcutsHelpOpen(false);
        } else if (zenMode) {
          setZenMode(false);
        } else {
          setSelectedNodeId(null);
        }
        return;
      }

      // F11 → Toggle zen mode
      if (key === 'f11') {
        event.preventDefault();
        setZenMode((prev) => !prev);
        return;
      }

      // Ctrl/Cmd+B → Toggle side panels
      if (mod && key === 'b') {
        event.preventDefault();
        setPanelsCollapsed((prev) => !prev);
        return;
      }
    };
    window.addEventListener('keydown', handler);
    return () => window.removeEventListener('keydown', handler);
  }, [handleFitView, handleSave, handleZoomIn, handleZoomOut, redo, shortcutsHelpOpen, undo, zenMode]);

  const handleRestoreVersion = useCallback(
    (snapshot: CanvasDocument) => {
      const restored = structuredClone(snapshot);
      setDoc(restored);
      setSelectedNodeId(null);
      setPast([]);
      setFuture([]);
      lastAutosavedRef.current = autosaveSignature(restored);
      toast.success(`Restored v${snapshot.version}. Save to keep it as the latest version.`);
    },
    [toast]
  );

  const handleTemplateSelect = async (templateDoc: CanvasDocument) => {
    const saved = await canvasStore.save(templateDoc);
    setDoc(saved);
    setSelectedNodeId(null);
    setPast([]);
    setFuture([]);
    navigate(`/canvas/${saved.id}`);
  };

  const deployValidation = useMemo(
    () =>
      doc
        ? validateCanvasForDeploy(doc, providerOptions, validatedProviders, installedCliProviders)
        : { ok: false, reason: 'Canvas is loading' },
    [doc, providerOptions, validatedProviders, installedCliProviders]
  );

  const handleDeploy = async () => {
    if (!doc || !deployValidation.ok) return;
    try {
      setReconciling(true);
      const saved = await canvasStore.save(structuredClone(doc));
      const deployed = await reconcileCanvas(saved.id);
      setDoc(deployed);
      setPast([]);
      setFuture([]);
      toast.success('Canvas deployed.');
    } catch (error) {
      toast.error(error instanceof Error ? error.message : 'Canvas deploy failed.');
      const refreshed = await canvasStore.get(doc.id);
      if (refreshed) setDoc(refreshed);
    } finally {
      setReconciling(false);
    }
  };

  const handleResume = async () => {
    if (!doc) return;
    try {
      setReconciling(true);
      const resumed = await reconcileCanvas(doc.id);
      setDoc(resumed);
      toast.success('Deploy resumed.');
    } catch (error) {
      toast.error(error instanceof Error ? error.message : 'Resume failed.');
      const refreshed = await canvasStore.get(doc.id);
      if (refreshed) setDoc(refreshed);
    } finally {
      setReconciling(false);
    }
  };

  const handleCancelDeploy = async () => {
    if (!doc) return;
    const cancelled = await cancelDeploy(doc.id);
    setDoc(cancelled);
    toast.warning('Deploy cancelled.');
  };

  const handleTearDown = async () => {
    if (!doc) return;
    try {
      setReconciling(true);
      const draft = await tearDownCanvas(doc.id);
      setDoc(draft);
      setPast([]);
      setFuture([]);
      toast.success('Canvas torn down.');
    } catch (error) {
      toast.error(error instanceof Error ? error.message : 'Tear Down failed.');
    } finally {
      setReconciling(false);
    }
  };

  if (!doc) {
    return (
      <Card glow="cyan">
        <h1 className="canvas-page-title">Canvas Builder</h1>
        <p>Loading canvas...</p>
      </Card>
    );
  }

  return (
    <main className={`canvas-builder-page${zenMode ? ' canvas-zen-mode canvas-fullscreen-mode' : ''}${panelsCollapsed ? ' canvas-panels-collapsed' : ''}`}>
      <header className="canvas-toolbar">
        <div>
          <h1 className="canvas-page-title">{doc.name}</h1>
          <span className="canvas-toolbar-meta">v{doc.version} - {doc.deploy_state.status}</span>
        </div>
        <div className="canvas-toolbar-actions">
          <Button variant="secondary" onClick={undo} disabled={past.length === 0}>
            Undo
          </Button>
          <Button variant="secondary" onClick={redo} disabled={future.length === 0}>
            Redo
          </Button>
          <Button variant="secondary" onClick={() => setTemplatePickerOpen(true)}>
            Use Template
          </Button>
          <Button variant="secondary" onClick={() => setHistoryOpen(true)}>
            History
          </Button>
          <Button variant="secondary" onClick={() => setPanelsCollapsed((p) => !p)} title="Toggle side panels (Ctrl+B)">
            {panelsCollapsed ? '⟨⟩ Panels' : '⟩⟨ Collapse'}
          </Button>
          <Button variant="secondary" onClick={() => setZenMode((p) => !p)} title="Zen mode — maximize canvas (F11)">
            {zenMode ? '⊡ Exit Zen' : '⊞ Zen Mode'}
          </Button>
          <Button variant="secondary" onClick={() => setVoiceOpen(true)}>
            Mic
          </Button>
          <Button variant="secondary" onClick={() => void handleSave()}>
            {isReconciling ? 'Reconciling...' : 'Save'}
          </Button>
          {doc.deploy_state.status === 'deploying' ? (
            <>
              <Button variant="secondary" onClick={() => void handleResume()} disabled={isReconciling}>
                Resume
              </Button>
              <Button variant="secondary" onClick={() => void handleCancelDeploy()} disabled={isReconciling}>
                Cancel
              </Button>
            </>
          ) : null}
          {doc.deploy_state.status === 'deployed' || doc.deploy_state.status === 'degraded' ? (
            <Button variant="secondary" onClick={() => void handleTearDown()} disabled={isReconciling}>
              Tear Down
            </Button>
          ) : null}
          {doc.deploy_state.status === 'degraded' ? (
            <Button
              variant="primary"
              disabled={!deployValidation.ok || isCanvasLocked}
              title={deployValidation.reason}
              onClick={() => void handleDeploy()}
            >
              Retry Failed
            </Button>
          ) : doc.deploy_state.status === 'draft' ? (
            <Button
              variant="primary"
              disabled={!deployValidation.ok || isCanvasLocked}
              title={deployValidation.reason}
              onClick={() => void handleDeploy()}
            >
              Deploy
            </Button>
          ) : null}
        </div>
      </header>

      <EdgeAdvisoryBanner visible={Boolean(doc.deploy_state.edge_change_advisory)} />

      {validatedProviders.length === 0 && installedCliProviders.length === 0 ? <NoProvidersNotice /> : null}
      {isReconciling ? (
        <Card className="canvas-reconciling-banner">Reconciling...</Card>
      ) : null}
      {isTouchOnly ? (
        <Card className="canvas-touch-banner">
          Editing on touch devices arrives in Milestone 2
        </Card>
      ) : null}

      <section className="canvas-workspace">
        {!isTouchOnly ? <AgentPalette /> : null}
        <div className="canvas-flow-shell" onDrop={handleDrop} onDragOver={(event) => event.preventDefault()}>
          <div className="canvas-floating-toolbar" aria-label="Canvas viewport controls">
            <button type="button" className="canvas-floating-control" onClick={handleFitView}>
              Fit View
            </button>
            <button type="button" className="canvas-floating-control" onClick={handleZoomOut} aria-label="Zoom out">
              -
            </button>
            <span className="canvas-zoom-level" aria-live="polite">
              {Math.round(zoomLevel * 100)}%
            </span>
            <button type="button" className="canvas-floating-control" onClick={handleZoomIn} aria-label="Zoom in">
              +
            </button>
            <button
              type="button"
              className="canvas-floating-control"
              onClick={() => setZenMode((prev) => !prev)}
              aria-pressed={zenMode}
            >
              {zenMode ? 'Exit Fullscreen' : 'Fullscreen'}
            </button>
            <button
              type="button"
              className="canvas-floating-control"
              onClick={() => setShortcutsHelpOpen(true)}
              aria-label="Show keyboard shortcuts"
              title="Keyboard shortcuts (?)"
            >
              ?
            </button>
          </div>
          {doc.nodes.length === 0 ? (
            <div className="canvas-empty-overlay">
              <span>Drop an agent block or start from a template.</span>
              <Button variant="secondary" onClick={() => setTemplatePickerOpen(true)}>
                Browse Templates
              </Button>
            </div>
          ) : null}
          <ReactFlow
            nodes={nodes}
            edges={edges}
            nodeTypes={nodeTypes}
            edgeTypes={edgeTypes}
            onNodesChange={onNodesChange}
            onEdgesChange={onEdgesChange}
            onConnect={onConnect}
            onReconnect={onReconnect}
            onReconnectStart={onReconnectStart}
            onReconnectEnd={onReconnectEnd}
            onNodeClick={(_event, node) => setSelectedNodeId(node.id)}
            onNodeContextMenu={onNodeContextMenu}
            onPaneClick={closeContextMenu}
            onSelectionChange={({ nodes: selectedNodes }) =>
              setSelectedNodeId(selectedNodes[0]?.id ?? null)
            }
            nodesDraggable={!isTouchOnly}
            nodesConnectable={!isTouchOnly && !isCanvasLocked}
            elementsSelectable={!isTouchOnly && !isCanvasLocked}
            deleteKeyCode={isCanvasLocked ? null : ['Backspace', 'Delete']}
            edgesReconnectable={!isTouchOnly && !isCanvasLocked}
            snapToGrid
            snapGrid={[24, 24]}
            fitView
            fitViewOptions={{ padding: 0.16, minZoom: 0.2, maxZoom: 1.4 }}
            minZoom={0.15}
            maxZoom={2.5}
            onlyRenderVisibleElements
            onViewportChange={(viewport) => setZoomLevel(viewport.zoom)}
            proOptions={{ hideAttribution: true }}
          >
            <Background color="rgba(0, 176, 189, 0.18)" gap={24} />
            <Controls />
            <MiniMap pannable zoomable />
            <div className="canvas-edge-legend">
              <div className="canvas-edge-legend-item">
                <span className="canvas-edge-legend-swatch canvas-edge-legend--handoff" />
                <span>handoff (solid)</span>
              </div>
              <div className="canvas-edge-legend-item">
                <span className="canvas-edge-legend-swatch canvas-edge-legend--assign" />
                <span>assign (dashed)</span>
              </div>
              <div className="canvas-edge-legend-item">
                <span className="canvas-edge-legend-swatch canvas-edge-legend--send_message" />
                <span>send_message (dotted)</span>
              </div>
              <div className="canvas-edge-legend-note">color = source agent</div>
            </div>
          </ReactFlow>
          {contextMenu && (
            <div
              ref={contextMenuRef}
              className="canvas-context-menu"
              style={{ top: contextMenu.y, left: contextMenu.x }}
            >
              <button
                className="canvas-context-menu-item"
                onClick={handleSetEntryPoint}
              >
                ⊎ Set as Entry Point
              </button>
              <button
                className="canvas-context-menu-item"
                onClick={handleDuplicateNode}
              >
                ⧉ Duplicate Node
              </button>
              <button
                className="canvas-context-menu-item canvas-context-menu-item--danger"
                onClick={handleDeleteNode}
              >
                ✕ Delete Node
              </button>
              <div className="canvas-context-menu-divider" />
              <button
                className="canvas-context-menu-item canvas-context-menu-item--danger"
                onClick={handleDeleteAllConnections}
              >
                ⊘ Delete All Connections
              </button>
            </div>
          )}
        </div>
        {!isTouchOnly ? (
          <BlockConfigurationPanel
            node={selectedNode}
            providerOptions={providerOptions}
            onUpdateNode={updateNode}
          />
        ) : null}
      </section>

      <TemplatePicker
        isOpen={isTemplatePickerOpen}
        onClose={() => setTemplatePickerOpen(false)}
        onSelect={(templateDoc) => void handleTemplateSelect(templateDoc)}
      />
      <KeyboardShortcutsHelp isOpen={shortcutsHelpOpen} onClose={closeShortcutsHelp} />
      <VersionHistory
        canvasId={doc.id}
        isOpen={historyOpen}
        onClose={() => setHistoryOpen(false)}
        onRestore={handleRestoreVersion}
      />
      <VoicePanel currentCanvas={doc} onUpdateCanvas={updateDoc} />
      <DeployProgressPanel status={doc.deploy_state.status} />
      <Modal
        isOpen={entryPointDialogOpen}
        onClose={() => setEntryPointDialogOpen(false)}
        title="Changing the entry point requires Tear Down"
        actions={
          <>
            <Button variant="secondary" onClick={() => setEntryPointDialogOpen(false)}>
              Cancel
            </Button>
            <Button
              variant="primary"
              onClick={() => {
                setEntryPointDialogOpen(false);
                void handleTearDown();
              }}
            >
              Tear Down
            </Button>
          </>
        }
      >
        Tear Down resets the deployed CAO session to draft before the entry point can change.
      </Modal>
    </main>
  );
};

function toFlowEdge(
  edge: CanvasEdge,
  nodes: CanvasNode[],
  onTypeChange: (edgeId: string, type: OrchestrationType) => void,
  onDelete: (edgeId: string) => void
): FlowEdge {
  const sourceNode = nodes.find((n) => n.id === edge.source);
  const sourceColor = sourceNode?.data.color;
  return {
    id: edge.id,
    source: edge.source,
    target: edge.target,
    type: 'orchestration',
    data: { orchestrationType: edge.type, onTypeChange, onDelete, sourceColor },
    markerEnd: { type: MarkerType.ArrowClosed, color: sourceColor || 'var(--cyan)' },
  };
}

function fromFlowEdge(edge: FlowEdge): CanvasEdge {
  const rawEdgeType = edge.data?.orchestrationType;
  const edgeType: OrchestrationType =
    rawEdgeType === 'assign' || rawEdgeType === 'send_message' || rawEdgeType === 'handoff'
      ? rawEdgeType
      : 'handoff';
  return {
    id: edge.id,
    source: edge.source,
    target: edge.target,
    type: edgeType,
    label: edgeType,
  };
}

export default CanvasBuilderPage;

function isEditableTarget(target: EventTarget | null): boolean {
  return (
    target instanceof HTMLElement &&
    (target.isContentEditable || ['INPUT', 'SELECT', 'TEXTAREA'].includes(target.tagName))
  );
}

// Signature of the meaningful canvas content. Excludes version/timestamps so
// autosave only fires on real edits, not on save-driven version bumps.
function autosaveSignature(doc: CanvasDocument): string {
  return JSON.stringify({
    name: doc.name,
    nodes: doc.nodes,
    edges: doc.edges,
    config: doc.config,
    deploy_state: doc.deploy_state,
  });
}
