declare module "@/lib/api/bindings/github.com/xuthus5/boxd/desktop/boxdbridgeservice" {
  export interface BridgeRequest {
    path: string
    body?: unknown
  }

  export interface BridgeResponse {
    data?: unknown
    error?: string
    status?: string
  }

  export function Call(req: BridgeRequest): Promise<BridgeResponse>
}

declare module "@/lib/api/bindings/github.com/xuthus5/boxd/desktop/boxdauthservice" {
  export interface EmbeddedSession {
    token: string
    expires_at: string
  }

  export function AutoLogin(): Promise<EmbeddedSession>
}
