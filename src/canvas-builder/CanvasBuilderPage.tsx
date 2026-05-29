/* eslint-disable agentverse/no-sideways-capability-imports */
import React, { useCallback, useEffect, useMemo, useState } from 'react';
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
  useReactFlow,
} from '@xyflow/react';
import '@xyflow/react/dist/style.css';
import { useNavigate, useParams } from 'react-router-dom';
import { Button, Card, Modal } from '@/design-system';
import { NoProvidersNotice } from '@/api/key-store/no-providers-notice';
import { useValidatedProviders } from '@/api/key-store/use-validated-providers';
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
import OrchestrationEdge from './OrchestrationEdge';
import { validateCanvasForDeploy } from './deploy-validation';
import { getCanvasProviderOptions } from './provider-options';
import { createAgentNode, StarterRole } from './role-templates';
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
  const providerOptions = useMemo(
    () => getCanvasProviderOptions(validatedProviders),
    [validatedProviders]
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

  useVoiceHotkey();


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
        setPast([]);
        setFuture([]);
      } else {
        const draft = canvasStore.createDraft();
        setDoc(draft);
        toast.warning('Canvas not found. Opened a new draft instead.');
      }
    }
    void loadCanvas();
    return () => {
      cancelled = true;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [id]);

  const updateDoc = useCallback(
    (updater: (current: CanvasDocument) => CanvasDocument) => {
      console.log('[DEBUG] updateDoc called, isReconciling:', isReconciling);
      setDoc((current) => {
        if (!current) {
          console.log('[DEBUG] updateDoc: current is null');
          return current;
        }
        if (isReconciling || current.deploy_state.status === 'deploying') {
          console.log('[DEBUG] updateDoc: early exit (isReconciling/deploying)', isReconciling, current.deploy_state.status);
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
        console.log('[DEBUG] updateDoc: updated successfully, nodes count:', nextDoc.nodes.length);
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
      const options = getCanvasProviderOptions(validatedProviders);
      console.log('[DEBUG] handleAddNodeEvent:', role, provider, 'validatedProviders:', validatedProviders, 'options:', options);
      const defaultProvider = provider || options[0]?.provider || 'claude_code';
      console.log('[DEBUG] handleAddNodeEvent defaultProvider:', defaultProvider);
      
      updateDoc((current) => {
        const newNode = createAgentNode({
          role: role as any,
          position: { x: 250, y: 150 + current.nodes.length * 50 },
          hasEntryPoint: current.nodes.some((n) => n.data.is_entry_point),
          provider: defaultProvider,
        });
        
        newNode.data.model = defaultProvider === 'claude_code' ? 'claude-3-5-sonnet' : 'gpt-4o';
        
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
  }, [updateDoc, validatedProviders, toast]);

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

  const nodes = useMemo<FlowNode[]>(
    () => {
      if (!doc || doc.nodes.length === 0) {
        return [
          {
            id: '__placeholder-agent__',
            type: 'agent',
            position: { x: 120, y: 140 },
            data: ({
              profile_name: 'placeholder',
              display_name: 'Placeholder Agent',
              role: 'custom',
              system_prompt: '',
              allowedTools: [],
              is_entry_point: false,
            } satisfies CanvasNode['data']) as unknown as Record<string, unknown>,
            draggable: false,
            connectable: false,
            selectable: false,
          },
        ];
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
    [doc?.nodes, isTouchOnly, selectedNodeId]
  );

  const edges = useMemo<FlowEdge[]>(
    () =>
      (doc?.edges ?? []).map((edge) => toFlowEdge(edge, handleEdgeTypeChange)),
    [doc?.edges, handleEdgeTypeChange]
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
          toFlowEdge(edge, handleEdgeTypeChange)
        );
        const nextFlowEdges = applyEdgeChanges(changes, currentFlowEdges);
        return {
          ...current,
          edges: nextFlowEdges.map(fromFlowEdge),
        };
      });
    },
    [isTouchOnly, isCanvasLocked, handleEdgeTypeChange, updateDoc]
  );

  const onConnect = useCallback(
    (connection: Connection) => {
      if (!connection.source || !connection.target || isTouchOnly || isCanvasLocked) return;
      updateDoc((current) => ({
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
      }));
    },
    [isTouchOnly, isCanvasLocked, updateDoc]
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
          }),
        ],
      }));
    },
    [isTouchOnly, reactFlow, updateDoc]
  );

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

  useEffect(() => {
    const handler = (event: KeyboardEvent) => {
      const isSave = (event.ctrlKey || event.metaKey) && event.key.toLowerCase() === 's';
      if (isSave) {
        event.preventDefault();
        void handleSave();
      }
    };
    window.addEventListener('keydown', handler);
    return () => window.removeEventListener('keydown', handler);
  }, [handleSave]);

  const undo = () => {
    setPast((items) => {
      const previous = items.at(-1);
      if (!previous || !doc) return items;
      setFuture((futureItems) => [structuredClone(doc), ...futureItems].slice(0, 20));
      setDoc(previous);
      return items.slice(0, -1);
    });
  };

  const redo = () => {
    setFuture((items) => {
      const next = items[0];
      if (!next || !doc) return items;
      setPast((pastItems) => [...pastItems.slice(-19), structuredClone(doc)]);
      setDoc(next);
      return items.slice(1);
    });
  };

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
        ? validateCanvasForDeploy(doc, providerOptions, validatedProviders)
        : { ok: false, reason: 'Canvas is loading' },
    [doc, providerOptions, validatedProviders]
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
    <main className="canvas-builder-page">
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

      {validatedProviders.length === 0 ? <NoProvidersNotice /> : null}
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
            onSelectionChange={({ nodes: selectedNodes }) =>
              setSelectedNodeId(selectedNodes[0]?.id ?? null)
            }
            nodesDraggable={!isTouchOnly}
            nodesConnectable={!isTouchOnly && !isCanvasLocked}
            elementsSelectable={!isTouchOnly && !isCanvasLocked}
            fitView
          >
            <Background color="rgba(0, 176, 189, 0.18)" gap={24} />
            <Controls />
            <MiniMap pannable zoomable />
          </ReactFlow>
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
  onTypeChange: (edgeId: string, type: OrchestrationType) => void
): FlowEdge {
  const dash = edge.type === 'assign' ? '8 6' : edge.type === 'send_message' ? '2 6' : undefined;
  return {
    id: edge.id,
    source: edge.source,
    target: edge.target,
    type: 'orchestration',
    data: { orchestrationType: edge.type, onTypeChange },
    markerEnd: { type: MarkerType.ArrowClosed, color: 'var(--cyan)' },
    style: {
      stroke: 'var(--cyan)',
      strokeDasharray: dash,
      transition: 'stroke-dasharray 80ms ease, opacity 80ms ease',
    },
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
