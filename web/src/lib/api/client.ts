import type { ApiError } from "./types"

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080"

export class ApiClientError extends Error {
  constructor(
    message: string,
    public status: number,
    public details?: ApiError
  ) {
    super(message)
    this.name = "ApiClientError"
  }
}

interface RequestOptions extends RequestInit {
  params?: Record<string, string | number | boolean | undefined>
}

async function request<T>(
  endpoint: string,
  options: RequestOptions = {}
): Promise<T> {
  const { params, ...fetchOptions } = options

  // Build URL with query params
  let url = `${API_BASE_URL}${endpoint}`
  if (params) {
    const searchParams = new URLSearchParams()
    Object.entries(params).forEach(([key, value]) => {
      if (value !== undefined) {
        searchParams.append(key, String(value))
      }
    })
    const queryString = searchParams.toString()
    if (queryString) {
      url += `?${queryString}`
    }
  }

  // Set default headers
  const headers = new Headers(fetchOptions.headers)
  if (!headers.has("Content-Type") && fetchOptions.body) {
    headers.set("Content-Type", "application/json")
  }

  const response = await fetch(url, {
    ...fetchOptions,
    headers,
  })

  // Handle non-OK responses
  if (!response.ok) {
    let errorDetails: ApiError | undefined
    try {
      errorDetails = await response.json()
    } catch {
      // Response body is not JSON
    }

    throw new ApiClientError(
      errorDetails?.detail || `HTTP ${response.status}: ${response.statusText}`,
      response.status,
      errorDetails
    )
  }

  // Handle empty responses
  if (response.status === 204) {
    return undefined as T
  }

  return response.json()
}

export const apiClient = {
  get<T>(endpoint: string, params?: RequestOptions["params"]): Promise<T> {
    return request<T>(endpoint, { method: "GET", params })
  },

  post<T>(endpoint: string, data?: unknown): Promise<T> {
    return request<T>(endpoint, {
      method: "POST",
      body: data ? JSON.stringify(data) : undefined,
    })
  },

  put<T>(endpoint: string, data?: unknown): Promise<T> {
    return request<T>(endpoint, {
      method: "PUT",
      body: data ? JSON.stringify(data) : undefined,
    })
  },

  patch<T>(endpoint: string, data?: unknown): Promise<T> {
    return request<T>(endpoint, {
      method: "PATCH",
      body: data ? JSON.stringify(data) : undefined,
    })
  },

  delete<T>(endpoint: string): Promise<T> {
    return request<T>(endpoint, { method: "DELETE" })
  },
}
