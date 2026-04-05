import { GraphState } from './graph-state.js';
import { CameraState } from './camera-state.js';
import { InteractionState } from './interaction-state.js';
import { AnimationState } from './animation-state.js';

export interface AppStoreConfig {
    mode?: 'edit' | 'view';
}

export class AppStore {
    graph: GraphState;
    camera: CameraState;
    interaction: InteractionState;
    animation: AnimationState;
    onChange: (() => void) | null = null;

    constructor(config?: AppStoreConfig) {
        this.graph = new GraphState();
        this.camera = new CameraState();
        this.interaction = new InteractionState();
        this.animation = new AnimationState();

        if (config?.mode) {
            this.interaction.mode = config.mode;
        }
    }

    notifyChange(): void {
        if (this.onChange) {
            this.onChange();
        }
    }
}
