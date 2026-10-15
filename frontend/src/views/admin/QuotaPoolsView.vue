<template>
  <AppLayout>
    <div class="space-y-6">
    <div class="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
      <div>
        <h1 class="text-2xl font-semibold text-gray-900 dark:text-gray-100">额度池管理</h1>
        <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
          将同类上游账号组成共享流量池，再按访问分组配置可用占比。
        </p>
      </div>
      <button class="btn btn-primary" @click="openCreatePool">新建额度池</button>
    </div>

    <div class="card overflow-hidden border border-gray-100 shadow-sm dark:border-gray-800">
      <div class="overflow-x-auto">
        <table class="min-w-full table-fixed divide-y divide-gray-200 text-sm dark:divide-gray-700">
          <colgroup>
            <col class="w-[36%]" />
            <col class="w-[15%]" />
            <col class="w-[13%]" />
            <col class="w-[20%]" />
            <col class="w-[16%]" />
          </colgroup>
          <thead class="bg-gray-50/80 text-xs uppercase tracking-wide text-gray-500 dark:bg-gray-800/60 dark:text-gray-400">
            <tr>
              <th class="px-6 py-3 text-left font-medium">名称</th>
              <th class="px-6 py-3 text-left font-medium">池总量</th>
              <th class="px-6 py-3 text-left font-medium">状态</th>
              <th class="px-6 py-3 text-left font-medium">更新时间</th>
              <th class="px-6 py-3 text-right font-medium">操作</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-gray-200 bg-white dark:divide-gray-700 dark:bg-gray-900">
            <tr v-if="loading">
              <td colspan="5" class="px-6 py-10 text-center text-sm text-gray-500">加载中...</td>
            </tr>
            <tr v-else-if="pools.length === 0">
              <td colspan="5" class="px-6 py-10 text-center text-sm text-gray-500">暂无额度池</td>
            </tr>
            <tr
              v-for="pool in pools"
              v-else
              :key="pool.id"
              class="transition-colors hover:bg-gray-50 dark:hover:bg-gray-800/60"
              :class="selectedPool?.id === pool.id ? 'bg-primary-50/60 dark:bg-primary-900/20' : ''"
            >
              <td class="px-6 py-4 text-left align-middle">
                <div class="truncate font-medium text-gray-900 dark:text-gray-100">{{ pool.name }}</div>
                <div v-if="pool.description" class="mt-1 truncate text-xs text-gray-500 dark:text-gray-400">{{ pool.description }}</div>
              </td>
              <td class="px-6 py-4 text-left align-middle">
                <span class="inline-flex items-center rounded-full bg-gray-100 px-2.5 py-1 text-xs font-medium text-gray-700 dark:bg-gray-800 dark:text-gray-300">100%</span>
              </td>
              <td class="px-6 py-4 text-left align-middle">
                <span :class="pool.enabled ? 'badge-success' : 'badge-muted'">{{ pool.enabled ? '启用' : '停用' }}</span>
              </td>
              <td class="truncate px-6 py-4 text-left align-middle text-gray-500 dark:text-gray-400">{{ formatDate(pool.updated_at || pool.created_at) }}</td>
              <td class="px-6 py-4 text-right align-middle">
                <div class="flex justify-end gap-2">
                  <button class="btn btn-secondary btn-sm" @click="selectPool(pool)">配置</button>
                  <button class="btn btn-secondary btn-sm" @click="openEditPool(pool)">编辑</button>
                  <button class="btn btn-danger btn-sm" @click="deletePool(pool)">删除</button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
      <div class="flex items-center justify-between border-t border-gray-200 px-4 py-3 text-sm dark:border-gray-700">
        <span class="text-gray-500">共 {{ total }} 条</span>
        <div class="flex items-center gap-2">
          <button class="btn btn-secondary btn-sm" :disabled="page <= 1" @click="changePage(page - 1)">上一页</button>
          <span class="text-gray-500">第 {{ page }} 页</span>
          <button class="btn btn-secondary btn-sm" :disabled="page * pageSize >= total" @click="changePage(page + 1)">下一页</button>
        </div>
      </div>
    </div>

    <div v-if="selectedPool" class="grid gap-6 xl:grid-cols-3">
      <section class="card p-5">
        <div class="mb-4 flex items-center justify-between">
          <div>
            <h2 class="text-lg font-semibold text-gray-900 dark:text-gray-100">总池账号</h2>
            <p class="text-sm text-gray-500">把多个同类上游账号加入同一个总池，整体按 100% 计算。</p>
          </div>
        </div>
        <div class="mb-4 rounded-lg bg-primary-50 p-3 text-sm text-primary-800 dark:bg-primary-900/20 dark:text-primary-200">
          当前总池：{{ poolAccounts.length }} 个账号共同组成 100%，分组占比从这个总池里扣减。
        </div>
        <div class="space-y-3">
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">添加上游账号到总池</label>
          <select v-model.number="accountForm.account_id" class="form-select w-full">
            <option :value="0">请选择要加入总池的账号</option>
            <option v-for="account in accountOptions" :key="account.id" :value="account.id">
              #{{ account.id }} {{ account.name || account.platform }}（{{ account.platform }}）
            </option>
          </select>
          <label class="flex items-center gap-2 text-sm text-gray-600 dark:text-gray-300">
            <input v-model="accountForm.enabled" type="checkbox" class="form-checkbox" /> 加入后启用
          </label>
          <button class="btn btn-primary w-full" :disabled="!accountForm.account_id || savingAccount" @click="addAccountBinding">
            加入总池
          </button>
        </div>
        <div class="mt-5 space-y-2">
          <div v-if="poolAccounts.length === 0" class="rounded-lg border border-dashed border-gray-300 p-4 text-center text-sm text-gray-500 dark:border-gray-700">
            暂无总池账号，请先把多个同类上游账号加入这个池。
          </div>
          <div v-for="binding in poolAccounts" :key="binding.id" class="flex items-center justify-between rounded-lg border border-gray-200 p-3 dark:border-gray-700">
            <div class="min-w-0">
              <div class="truncate text-sm font-medium text-gray-900 dark:text-gray-100">{{ accountLabel(binding.account_id, binding.platform) }}</div>
              <div class="text-xs text-gray-500">{{ binding.enabled ? '启用' : '停用' }} · 快照 {{ formatPercentValue(binding.last_usage_percent) }}%</div>
            </div>
            <button class="btn btn-danger btn-sm" @click="removeAccountBinding(binding)">移除</button>
          </div>
        </div>
      </section>

      <section class="card p-5">
        <div class="mb-4">
          <h2 class="text-lg font-semibold text-gray-900 dark:text-gray-100">分组占比</h2>
          <p class="text-sm text-gray-500">额度池整体按 100% 计算，访问分组只配置可用占比。</p>
        </div>
        <div class="space-y-3">
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">访问分组</label>
          <select v-model.number="limitForm.group_id" class="form-select w-full">
            <option :value="0">请选择分组</option>
            <option v-for="group in groups" :key="group.id" :value="group.id">{{ group.name }}（{{ group.platform }}）</option>
          </select>
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">占比（%）</label>
          <input v-model.number="limitForm.percent" type="number" min="0" max="100" step="0.01" class="form-input w-full" />
          <label class="flex items-center gap-2 text-sm text-gray-600 dark:text-gray-300">
            <input v-model="limitForm.enabled" type="checkbox" class="form-checkbox" /> 启用限制
          </label>
          <button class="btn btn-primary w-full" :disabled="!limitForm.group_id || savingLimit" @click="saveGroupLimit">
            保存占比
          </button>
        </div>
        <div class="mt-5 space-y-2">
          <div v-if="groupLimits.length === 0" class="rounded-lg border border-dashed border-gray-300 p-4 text-center text-sm text-gray-500 dark:border-gray-700">
            暂未配置分组占比
          </div>
          <div v-for="limit in groupLimits" :key="limit.id" class="rounded-lg border border-gray-200 p-3 dark:border-gray-700">
            <div class="flex items-center justify-between gap-3">
              <div>
                <div class="text-sm font-medium text-gray-900 dark:text-gray-100">{{ groupName(limit.group_id) }}</div>
                <div class="text-xs text-gray-500">
                  {{ basisPointsToPercent(limit.percent_basis_points) }}% · {{ limit.enabled ? '启用' : '停用' }}
                </div>
              </div>
              <div class="flex gap-2">
                <button class="btn btn-secondary btn-sm" @click="editGroupLimit(limit)">编辑</button>
                <button class="btn btn-danger btn-sm" @click="removeGroupLimit(limit)">删除</button>
              </div>
            </div>
          </div>
        </div>
      </section>

      <section class="card p-5">
        <div class="mb-4 flex items-center justify-between">
          <div>
            <h2 class="text-lg font-semibold text-gray-900 dark:text-gray-100">当前用量</h2>
            <p class="text-sm text-gray-500">按总池汇总展示各分组已用百分比。</p>
          </div>
          <button class="btn btn-secondary btn-sm" @click="refreshSelectedPool">刷新</button>
        </div>
        <div class="space-y-2">
          <div v-if="groupUsage.length === 0" class="rounded-lg border border-dashed border-gray-300 p-4 text-center text-sm text-gray-500 dark:border-gray-700">
            暂无用量记录
          </div>
          <div v-for="usage in groupUsage" :key="usage.id" class="rounded-lg border border-gray-200 p-3 dark:border-gray-700">
            <div class="flex items-center justify-between gap-3">
              <div class="min-w-0 flex-1">
                <div class="flex items-center justify-between text-sm">
                  <span class="font-medium text-gray-900 dark:text-gray-100">{{ groupName(usage.group_id) }}</span>
                  <span class="text-gray-500">{{ formatPercentValue(usage.used_cost) }}%</span>
                </div>
                <div class="mt-2 h-2 overflow-hidden rounded-full bg-gray-100 dark:bg-gray-800">
                  <div class="h-full rounded-full bg-primary-500" :style="{ width: `${usagePercent(usage)}%` }"></div>
                </div>
                <div class="mt-1 text-xs text-gray-500">上限 {{ groupLimitPercent(usage.group_id) }}%</div>
              </div>
              <button class="btn btn-secondary btn-sm" @click="resetUsage(usage)">重置</button>
            </div>
          </div>
        </div>
      </section>
    </div>

    <div v-if="showPoolModal" class="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4">
      <div class="w-full max-w-lg rounded-xl bg-white p-6 shadow-xl dark:bg-gray-900">
        <div class="mb-5 flex items-center justify-between">
          <h2 class="text-lg font-semibold text-gray-900 dark:text-gray-100">{{ editingPool ? '编辑额度池' : '新建额度池' }}</h2>
          <button class="text-gray-400 hover:text-gray-600" @click="showPoolModal = false">×</button>
        </div>
        <div class="space-y-4">
          <div>
            <label class="form-label">名称</label>
            <input v-model.trim="poolForm.name" class="form-input w-full" placeholder="例如 GPT 总池" />
          </div>
          <div>
            <label class="form-label">描述</label>
            <textarea v-model.trim="poolForm.description" class="form-textarea w-full" rows="3" placeholder="可选"></textarea>
          </div>
          <label class="flex items-center gap-2 text-sm text-gray-600 dark:text-gray-300">
            <input v-model="poolForm.enabled" type="checkbox" class="form-checkbox" /> 启用额度池
          </label>
        </div>
        <div class="mt-6 flex justify-end gap-3">
          <button class="btn btn-secondary" @click="showPoolModal = false">取消</button>
          <button class="btn btn-primary" :disabled="!poolForm.name || savingPool" @click="savePool">保存</button>
        </div>
      </div>
    </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { adminAPI } from '@/api/admin'
import AppLayout from '@/components/layout/AppLayout.vue'
import type { AccountListItem, AdminGroup } from '@/types'
import type {
  QuotaPool,
  QuotaPoolAccount,
  QuotaPoolGroupLimit,
  QuotaPoolGroupUsage
} from '@/api/admin/quotaPools'

const page = ref(1)
const pageSize = 20
const total = ref(0)
const loading = ref(false)
const pools = ref<QuotaPool[]>([])
const selectedPool = ref<QuotaPool | null>(null)

const accounts = ref<AccountListItem[]>([])
const groups = ref<AdminGroup[]>([])
const poolAccounts = ref<QuotaPoolAccount[]>([])
const groupLimits = ref<QuotaPoolGroupLimit[]>([])
const groupUsage = ref<QuotaPoolGroupUsage[]>([])

const showPoolModal = ref(false)
const editingPool = ref<QuotaPool | null>(null)
const savingPool = ref(false)
const savingAccount = ref(false)
const savingLimit = ref(false)

const poolForm = reactive({
  name: '',
  description: '',
  enabled: true
})

const accountForm = reactive({
  account_id: 0,
  enabled: true
})

const limitForm = reactive({
  group_id: 0,
  percent: 0,
  enabled: true
})

const accountOptions = computed(() => accounts.value.filter((account) => !poolAccounts.value.some((binding) => binding.account_id === account.id)))

onMounted(async () => {
  await Promise.all([loadPools(), loadLookups()])
})

async function loadPools() {
  loading.value = true
  try {
    const result = await adminAPI.quotaPools.list(page.value, pageSize)
    pools.value = result.items || []
    total.value = result.total || 0
    if (selectedPool.value) {
      const updated = pools.value.find((pool) => pool.id === selectedPool.value?.id)
      if (updated) selectedPool.value = updated
    }
  } finally {
    loading.value = false
  }
}

async function loadLookups() {
  const [accountResult, groupResult] = await Promise.all([
    adminAPI.accounts.list(1, 500, { lite: '1' }),
    adminAPI.groups.getAllIncludingInactive()
  ])
  accounts.value = accountResult.items || []
  groups.value = groupResult || []
}

async function changePage(next: number) {
  page.value = next
  await loadPools()
}

function openCreatePool() {
  editingPool.value = null
  Object.assign(poolForm, {
    name: '',
    description: '',
    enabled: true
  })
  showPoolModal.value = true
}

function openEditPool(pool: QuotaPool) {
  editingPool.value = pool
  Object.assign(poolForm, {
    name: pool.name,
    description: pool.description || '',
    enabled: pool.enabled
  })
  showPoolModal.value = true
}

async function savePool() {
  savingPool.value = true
  try {
    const payload = {
      name: poolForm.name,
      description: poolForm.description,
      enabled: poolForm.enabled,
      default_action: 'allow'
    }
    if (editingPool.value) {
      await adminAPI.quotaPools.update(editingPool.value.id, payload)
    } else {
      await adminAPI.quotaPools.create(payload)
    }
    showPoolModal.value = false
    await loadPools()
  } finally {
    savingPool.value = false
  }
}

async function deletePool(pool: QuotaPool) {
  if (!window.confirm(`确定删除额度池「${pool.name}」吗？`)) return
  await adminAPI.quotaPools.remove(pool.id)
  if (selectedPool.value?.id === pool.id) selectedPool.value = null
  await loadPools()
}

async function selectPool(pool: QuotaPool) {
  selectedPool.value = pool
  resetForms()
  await refreshSelectedPool()
}

async function refreshSelectedPool() {
  if (!selectedPool.value) return
  const poolId = selectedPool.value.id
  const [accountsResult, limitsResult, usageResult] = await Promise.all([
    adminAPI.quotaPools.listAccounts(poolId),
    adminAPI.quotaPools.listGroupLimits(poolId),
    adminAPI.quotaPools.listGroupUsage(poolId)
  ])
  poolAccounts.value = accountsResult || []
  groupLimits.value = limitsResult || []
  groupUsage.value = usageResult || []
}

async function addAccountBinding() {
  if (!selectedPool.value || !accountForm.account_id) return
  const account = accounts.value.find((item) => item.id === accountForm.account_id)
  if (!account) return

  savingAccount.value = true
  try {
    await adminAPI.quotaPools.addAccount(selectedPool.value.id, {
      account_id: account.id,
      platform: account.platform,
      enabled: accountForm.enabled
    })
    accountForm.account_id = 0
    accountForm.enabled = true
    await refreshSelectedPool()
  } finally {
    savingAccount.value = false
  }
}

async function removeAccountBinding(binding: QuotaPoolAccount) {
  if (!selectedPool.value) return
  await adminAPI.quotaPools.deleteAccountBinding(selectedPool.value.id, binding.id)
  await refreshSelectedPool()
}

async function saveGroupLimit() {
  if (!selectedPool.value || !limitForm.group_id) return
  savingLimit.value = true
  try {
    await adminAPI.quotaPools.upsertGroupLimit(selectedPool.value.id, limitForm.group_id, {
      percent_basis_points: Math.round(Number(limitForm.percent || 0) * 100),
      enabled: limitForm.enabled
    })
    limitForm.group_id = 0
    limitForm.percent = 0
    limitForm.enabled = true
    await refreshSelectedPool()
  } finally {
    savingLimit.value = false
  }
}

function editGroupLimit(limit: QuotaPoolGroupLimit) {
  limitForm.group_id = limit.group_id
  limitForm.percent = basisPointsToPercent(limit.percent_basis_points)
  limitForm.enabled = limit.enabled
}

async function removeGroupLimit(limit: QuotaPoolGroupLimit) {
  if (!selectedPool.value) return
  await adminAPI.quotaPools.deleteGroupLimit(selectedPool.value.id, limit.group_id)
  await refreshSelectedPool()
}

async function resetUsage(usage: QuotaPoolGroupUsage) {
  if (!selectedPool.value) return
  if (!window.confirm(`确定重置「${groupName(usage.group_id)}」当前用量吗？`)) return
  await adminAPI.quotaPools.resetGroupUsage(selectedPool.value.id, {
    group_id: usage.group_id
  })
  await refreshSelectedPool()
}

function resetForms() {
  Object.assign(accountForm, { account_id: 0, enabled: true })
  Object.assign(limitForm, { group_id: 0, percent: 0, enabled: true })
}

function formatPercentValue(value?: number | null) {
  return Number(value || 0).toFixed(2)
}

function formatDate(value?: string) {
  if (!value) return '-'
  return new Date(value).toLocaleString()
}

function accountLabel(accountId: number, fallbackPlatform: string) {
  const account = accounts.value.find((item) => item.id === accountId)
  if (!account) return `#${accountId}（${fallbackPlatform}）`
  return `#${account.id} ${account.name || account.platform}（${account.platform}）`
}

function groupName(groupId: number) {
  return groups.value.find((group) => group.id === groupId)?.name || `分组 #${groupId}`
}

function basisPointsToPercent(value: number) {
  return Number((value / 100).toFixed(2))
}

function groupLimitPercent(groupId: number) {
  const limit = groupLimits.value.find((item) => item.group_id === groupId)
  if (!limit) return '0.00'
  return basisPointsToPercent(limit.percent_basis_points).toFixed(2)
}

function usagePercent(usage: QuotaPoolGroupUsage) {
  const limit = groupLimits.value.find((item) => item.group_id === usage.group_id)
  if (!limit || limit.percent_basis_points <= 0) return 0
  const max = limit.percent_basis_points / 100
  return Math.min(100, (Number(usage.used_cost || 0) / max) * 100)
}
</script>
