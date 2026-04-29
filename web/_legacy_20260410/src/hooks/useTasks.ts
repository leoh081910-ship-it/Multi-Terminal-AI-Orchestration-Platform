import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { CreateTaskInput, Task, TaskStats } from '../types'

const API_URL = '/api'

type APIResponse<T> = {
  success: boolean
  data?: T
  error?: string
}

async function readAPIResponse<T>(res: Response): Promise<T> {
  const payload = (await res.json()) as APIResponse<T>

  if (!res.ok || !payload.success || payload.data === undefined) {
    throw new Error(payload.error || 'Request failed')
  }

  return payload.data
}

export function useTasks() {
  return useQuery({
    queryKey: ['tasks'],
    queryFn: async (): Promise<Task[]> => {
      const res = await fetch(`${API_URL}/tasks/`)
      return readAPIResponse<Task[]>(res)
    },
  })
}

export function useTaskStats() {
  return useQuery({
    queryKey: ['tasks', 'stats'],
    queryFn: async (): Promise<TaskStats> => {
      const res = await fetch(`${API_URL}/tasks/stats`)
      return readAPIResponse<TaskStats>(res)
    },
  })
}

export function useCreateTask() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (input: CreateTaskInput): Promise<{ id: string }> => {
      const res = await fetch(`${API_URL}/tasks/`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(input),
      })
      return readAPIResponse<{ id: string }>(res)
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['tasks'] })
      queryClient.invalidateQueries({ queryKey: ['tasks', 'stats'] })
    },
  })
}

export function useRetryTask() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (taskId: string): Promise<{ id: string; status: string }> => {
      const res = await fetch(`${API_URL}/tasks/${taskId}/retry`, {
        method: 'POST',
      })
      return readAPIResponse<{ id: string; status: string }>(res)
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['tasks'] })
      queryClient.invalidateQueries({ queryKey: ['tasks', 'stats'] })
    },
  })
}
