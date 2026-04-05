import type { Graph, Node, NodeType } from '../core/types.js';

export interface AppConfig {
    apiBase: string;
    mode: string;
}

export class ApiClient {
    private baseUrl: string;

    constructor(baseUrl = '/api') {
        this.baseUrl = baseUrl;
    }

    private async request<T>(path: string, init?: RequestInit): Promise<T> {
        const res = await fetch(`${this.baseUrl}${path}`, {
            headers: { 'Content-Type': 'application/json' },
            ...init,
        });
        if (!res.ok) {
            throw new Error(`API error: ${res.status} ${res.statusText}`);
        }
        if (res.status === 204) {
            return undefined as T;
        }
        return res.json() as Promise<T>;
    }

    async getConfig(): Promise<AppConfig> {
        return this.request<AppConfig>('/config');
    }

    async listGraphs(): Promise<string[]> {
        return this.request<string[]>('/graphs');
    }

    async getGraph(id: string): Promise<Graph> {
        return this.request<Graph>(`/graphs/${id}`);
    }

    async createGraph(graph: Graph): Promise<Graph> {
        return this.request<Graph>('/graphs', {
            method: 'POST',
            body: JSON.stringify(graph),
        });
    }

    async updateGraph(id: string, graph: Graph): Promise<Graph> {
        return this.request<Graph>(`/graphs/${id}`, {
            method: 'PUT',
            body: JSON.stringify(graph),
        });
    }

    async deleteGraph(id: string): Promise<void> {
        return this.request<void>(`/graphs/${id}`, {
            method: 'DELETE',
        });
    }

    async addNode(graphId: string, node: Node): Promise<Node> {
        return this.request<Node>(`/graphs/${graphId}/nodes`, {
            method: 'POST',
            body: JSON.stringify(node),
        });
    }

    async getNodeTypes(): Promise<NodeType[]> {
        return this.request<NodeType[]>('/node-types');
    }

    async executeGraph(id: string): Promise<void> {
        return this.request<void>(`/graphs/${id}/execute`, {
            method: 'POST',
        });
    }

    async clickNode(graphId: string, nodeId: string): Promise<Node> {
        return this.request<Node>(`/graphs/${graphId}/nodes/${nodeId}/click`, {
            method: 'POST',
        });
    }
}
