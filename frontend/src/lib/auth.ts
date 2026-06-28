import { useQuery } from '@tanstack/react-query'
import { api, ApiError } from './api'

// useMe loads the current user. A 401 means "not logged in" and is not retried.
export function useMe() {
  return useQuery({
    queryKey: ['me'],
    queryFn: api.getMe,
    retry: (count, err) =>
      !(err instanceof ApiError && err.status === 401) && count < 2,
    staleTime: 60_000,
  })
}
