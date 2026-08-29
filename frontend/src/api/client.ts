// ApiError carries the HTTP status so callers can branch: 404 means "does
// not exist, don't offer retry," anything else means "transient, offer
// retry." See docs/plans/atlas-frontend/02-architecture.md.
export class ApiError extends Error {
  status: number;

  constructor(status: number, message: string) {
    super(message);
    this.name = "ApiError";
    this.status = status;
  }
}

export async function apiGet<T>(
  path: string,
  params?: Record<string, string>,
): Promise<T> {
  let url = path;
  if (params && Object.keys(params).length > 0) {
    url += `?${new URLSearchParams(params).toString()}`;
  }

  const res = await fetch(url);
  if (!res.ok) {
    let message = res.statusText;
    try {
      const body = (await res.json()) as { error?: string };
      if (body.error) message = body.error;
    } catch {
      // Non-JSON error body, fall back to statusText.
    }
    throw new ApiError(res.status, message);
  }
  return (await res.json()) as T;
}
