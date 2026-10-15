import { apiClient } from '../client'
import type { PaginatedResponse } from '@/types'

export interface QuotaPool {
  id: number
  name: string
  description?: string | null
  enabled: boolean
  period_type: string
  default_action: string
  created_at?: string
  updated_at?: string
}

export interface QuotaPoolAccount {
  id: number
  quota_pool_id: number
  platform: string
  account_id: number
  enabled: boolean
  last_usage_percent: number
  created_at?: string
  updated_at?: string
}

export interface QuotaPoolGroupLimit {
  id: number
  quota_pool_id: number
  group_id: number
  percent_basis_points: number
  enabled: boolean
  created_at?: string
  updated_at?: string
}

export interface QuotaPoolGroupUsage {
  id: number
  quota_pool_id: number
  quota_pool_account_id?: number
  group_id: number
  period_type: string
  period_key: string
  used_cost: number
  created_at?: string
  updated_at?: string
}

export interface SaveQuotaPoolRequest {
  name: string
  description?: string
  enabled: boolean
  default_action?: string
}

export interface AddQuotaPoolAccountRequest {
  platform: string
  account_id: number
  enabled: boolean
}

export interface UpsertQuotaPoolGroupLimitRequest {
  percent_basis_points: number
  enabled: boolean
}

export interface ResetQuotaPoolUsageRequest {
  group_id: number
}

export async function list(page = 1, pageSize = 20): Promise<PaginatedResponse<QuotaPool>> {
  const { data } = await apiClient.get<PaginatedResponse<QuotaPool>>('/admin/quota-pools', {
    params: { page, page_size: pageSize }
  })
  return data
}

export async function listAll(): Promise<QuotaPool[]> {
  const { data } = await apiClient.get<QuotaPool[]>('/admin/quota-pools/all')
  return data
}

export async function getById(id: number): Promise<QuotaPool> {
  const { data } = await apiClient.get<QuotaPool>(`/admin/quota-pools/${id}`)
  return data
}

export async function create(payload: SaveQuotaPoolRequest): Promise<QuotaPool> {
  const { data } = await apiClient.post<QuotaPool>('/admin/quota-pools', payload)
  return data
}

export async function update(id: number, payload: SaveQuotaPoolRequest): Promise<QuotaPool> {
  const { data } = await apiClient.put<QuotaPool>(`/admin/quota-pools/${id}`, payload)
  return data
}

export async function remove(id: number): Promise<void> {
  await apiClient.delete(`/admin/quota-pools/${id}`)
}

export async function listAccounts(poolId: number): Promise<QuotaPoolAccount[]> {
  const { data } = await apiClient.get<QuotaPoolAccount[]>(`/admin/quota-pools/${poolId}/accounts`)
  return data
}

export async function addAccount(poolId: number, payload: AddQuotaPoolAccountRequest): Promise<QuotaPoolAccount> {
  const { data } = await apiClient.post<QuotaPoolAccount>(`/admin/quota-pools/${poolId}/accounts`, payload)
  return data
}

export async function deleteAccountBinding(poolId: number, bindingId: number): Promise<void> {
  await apiClient.delete(`/admin/quota-pools/${poolId}/accounts/${bindingId}`)
}

export async function listGroupLimits(poolId: number): Promise<QuotaPoolGroupLimit[]> {
  const { data } = await apiClient.get<QuotaPoolGroupLimit[]>(`/admin/quota-pools/${poolId}/group-limits`)
  return data
}

export async function upsertGroupLimit(
  poolId: number,
  groupId: number,
  payload: UpsertQuotaPoolGroupLimitRequest
): Promise<QuotaPoolGroupLimit> {
  const { data } = await apiClient.put<QuotaPoolGroupLimit>(`/admin/quota-pools/${poolId}/group-limits/${groupId}`, payload)
  return data
}

export async function deleteGroupLimit(poolId: number, groupId: number): Promise<void> {
  await apiClient.delete(`/admin/quota-pools/${poolId}/group-limits/${groupId}`)
}

export async function listGroupUsage(poolId: number): Promise<QuotaPoolGroupUsage[]> {
  const { data } = await apiClient.get<QuotaPoolGroupUsage[]>(`/admin/quota-pools/${poolId}/group-usage`)
  return data
}

export async function resetGroupUsage(poolId: number, payload: ResetQuotaPoolUsageRequest): Promise<void> {
  await apiClient.post(`/admin/quota-pools/${poolId}/group-usage/reset`, payload)
}

export const quotaPoolsAPI = {
  list,
  listAll,
  getById,
  create,
  update,
  remove,
  listAccounts,
  addAccount,
  deleteAccountBinding,
  listGroupLimits,
  upsertGroupLimit,
  deleteGroupLimit,
  listGroupUsage,
  resetGroupUsage
}

export default quotaPoolsAPI
