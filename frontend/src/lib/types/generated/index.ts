// Generated types from backend service definitions
export type ThreadStatus = 'idle' | 'running' | 'completed' | 'error';

export interface Thread {
  id: string;
  title: string;
  goal?: string;
  status?: ThreadStatus;
  createdAt: string;
  updatedAt: string;
  deletedAt?: string | null;
}
