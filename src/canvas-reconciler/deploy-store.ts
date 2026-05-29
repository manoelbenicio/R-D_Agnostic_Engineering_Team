import { create } from 'zustand';

export interface DeployStep {
  id: string;
  label: string;
  status: 'pending' | 'in_flight' | 'success' | 'failed';
}

export interface DeployProgressState {
  activeDeployCanvasId: string | null;
  deploySteps: DeployStep[];
  startDeploy: (canvasId: string, steps: DeployStep[]) => void;
  updateStepStatus: (stepId: string, status: DeployStep['status']) => void;
  clearDeploy: () => void;
}

export const useDeployStore = create<DeployProgressState>((set) => ({
  activeDeployCanvasId: null,
  deploySteps: [],
  startDeploy: (canvasId, steps) => set({ activeDeployCanvasId: canvasId, deploySteps: steps }),
  updateStepStatus: (stepId, status) => set((state) => ({
    deploySteps: state.deploySteps.map(step => 
      step.id === stepId ? { ...step, status } : step
    )
  })),
  clearDeploy: () => set({ activeDeployCanvasId: null, deploySteps: [] })
}));
