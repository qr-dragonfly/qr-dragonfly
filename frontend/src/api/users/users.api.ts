import { requestJson } from '../http'
import { emitAuthChanged } from '../../lib/authEvents'
import type {
  AuthSession,
  ChangePasswordInput,
  ConfirmForgotPasswordInput,
  ConfirmSignUpInput,
  CreateUserInput,
  ForgotPasswordInput,
  LoginInput,
  ResendConfirmationInput,
  StatusResponse,
  UpdateUserInput,
  User,
} from './users.types'

export const usersApi = {
  // Account creation
  async register(input: CreateUserInput): Promise<AuthSession> {
    const session = await requestJson<AuthSession>({
      method: 'POST',
      path: '/api/users/register',
      body: input,
      credentials: 'include',
    })

    emitAuthChanged()
    return session
  },

  // Auth
  async login(input: LoginInput): Promise<AuthSession> {
    const session = await requestJson<AuthSession>({
      method: 'POST',
      path: '/api/users/login',
      body: input,
      credentials: 'include',
    })

    emitAuthChanged()
    return session
  },

  confirmSignUp(input: ConfirmSignUpInput): Promise<StatusResponse> {
    return requestJson<StatusResponse>({
      method: 'POST',
      path: '/api/users/confirm',
      body: input,
      credentials: 'include',
    })
  },

  resendConfirmation(input: ResendConfirmationInput): Promise<StatusResponse> {
    return requestJson<StatusResponse>({
      method: 'POST',
      path: '/api/users/resend-confirmation',
      body: input,
      credentials: 'include',
    })
  },

  forgotPassword(input: ForgotPasswordInput): Promise<StatusResponse> {
    return requestJson<StatusResponse>({
      method: 'POST',
      path: '/api/users/forgot-password',
      body: input,
      credentials: 'include',
    })
  },

  confirmForgotPassword(input: ConfirmForgotPasswordInput): Promise<StatusResponse> {
    return requestJson<StatusResponse>({
      method: 'POST',
      path: '/api/users/confirm-forgot-password',
      body: input,
      credentials: 'include',
    })
  },

  changePassword(input: ChangePasswordInput): Promise<StatusResponse> {
    return requestJson<StatusResponse>({
      method: 'POST',
      path: '/api/users/change-password',
      body: input,
      credentials: 'include',
    })
  },

  async logout(): Promise<void> {
    await requestJson<void>({
      method: 'POST',
      path: '/api/users/logout',
      credentials: 'include',
    })

    emitAuthChanged()
  },

  me(): Promise<User> {
    return requestJson<User>({
      method: 'GET',
      path: '/api/users/me',
      credentials: 'include',
    })
  },

  // Admin-style CRUD (if your backend supports it)
  list(): Promise<User[]> {
    return requestJson<User[]>({
      method: 'GET',
      path: '/api/users',
      credentials: 'include',
    })
  },

  getById(id: string): Promise<User> {
    return requestJson<User>({
      method: 'GET',
      path: `/api/users/${encodeURIComponent(id)}`,
      credentials: 'include',
    })
  },

  create(input: CreateUserInput): Promise<User> {
    return requestJson<User>({
      method: 'POST',
      path: '/api/users',
      body: input,
      credentials: 'include',
    })
  },

  update(id: string, patch: UpdateUserInput): Promise<User> {
    return requestJson<User>({
      method: 'PATCH',
      path: `/api/users/${encodeURIComponent(id)}`,
      body: patch,
      credentials: 'include',
    })
  },

  delete(id: string): Promise<void> {
    return requestJson<void>({
      method: 'DELETE',
      path: `/api/users/${encodeURIComponent(id)}`,
      credentials: 'include',
    })
  },
}
