<script setup>
import Button from "@/components/ui/Button.vue";
import Card from "@/components/ui/Card.vue";
import Input from "@/components/ui/Input.vue";
import Select from "@/components/ui/Select.vue";
import { appState, syncStats } from "@/state/appState";
import { formatCompactInteger, formatInteger } from "@/utils/numberFormat";
import { computed, onMounted, ref, watch } from "vue";

const TOKEN_PRICE_PER_MILLION = {
  input: 5,
  output: 25,
  cacheRead: 0.5,
  cacheWrite: 6.25,
};

const PERIOD_OPTIONS = [
  { label: "今天", value: "today" },
  { label: "昨天", value: "yesterday" },
  { label: "近 24 小时", value: "last24h" },
  { label: "近 7 天", value: "last7d" },
  { label: "近 14 天", value: "last14d" },
  { label: "近 30 天", value: "last30d" },
  { label: "本月", value: "thisMonth" },
  { label: "上月", value: "lastMonth" },
  { label: "自定义时间段", value: "custom" },
];

const SORT_OPTIONS = [
  { label: "按 Token", value: "token" },
  { label: "按价格估算", value: "cost" },
];

const period = ref("last7d");
const customStart = ref("");
const customEnd = ref("");
const selectedModels = ref([]);
const sortBy = ref("token");

const modelOptions = computed(() => {
  const seen = new Set();
  const options = [];
  for (const adapter of appState.modelAdapters || []) {
    const value = adapter && adapter.modelID ? String(adapter.modelID) : "";
    if (!value || seen.has(value)) {
      continue;
    }
    seen.add(value);
    options.push({
      label: adapter.displayName || adapter.modelID,
      value,
    });
  }
  return options;
});

const showCustomRange = computed(() => period.value === "custom");
const hasUnknownModel = computed(() =>
  appState.stats.byModel.some((bucket) => bucket.key === "unknown"),
);

function normalizeNumber(value) {
  const number = Number(value);
  if (!Number.isFinite(number)) {
    return 0;
  }
  return Math.round(number);
}

function priceTokens(tokens, pricePerMillion) {
  return (normalizeNumber(tokens) / 1_000_000) * pricePerMillion;
}

function bucketCost(bucket) {
  const input = priceTokens(bucket.inputTokens, TOKEN_PRICE_PER_MILLION.input);
  const output = priceTokens(bucket.outputTokens, TOKEN_PRICE_PER_MILLION.output);
  const cacheRead = priceTokens(bucket.cacheReadTokens, TOKEN_PRICE_PER_MILLION.cacheRead);
  const cacheWrite = priceTokens(bucket.cacheWriteTokens, TOKEN_PRICE_PER_MILLION.cacheWrite);
  return input + output + cacheRead + cacheWrite;
}

function formatUSD(value) {
  const amount = Number(value);
  if (!Number.isFinite(amount)) {
    return "$0.00";
  }
  if (amount > 0 && amount < 0.01) {
    return "<$0.01";
  }
  return `$${amount.toFixed(2)}`;
}

function modelLabel(value) {
  if (value === "unknown") {
    return "未知/其他";
  }
  const matched = modelOptions.value.find((option) => option.value === value);
  return matched ? matched.label : value;
}

function sortBuckets(buckets) {
  const list = buckets.slice();
  list.sort((left, right) => {
    if (sortBy.value === "cost") {
      return bucketCost(right) - bucketCost(left);
    }
    return right.totalTokens - left.totalTokens;
  });
  return list;
}

const sortedByDay = computed(() => sortBuckets(appState.stats.byDay));
const sortedByModel = computed(() => sortBuckets(appState.stats.byModel));
const total = computed(() => appState.stats.total);
const totalCost = computed(() => bucketCost(appState.stats.total));

function toggleModel(value) {
  const index = selectedModels.value.indexOf(value);
  if (index >= 0) {
    selectedModels.value = selectedModels.value.filter((item) => item !== value);
    return;
  }
  selectedModels.value = [...selectedModels.value, value];
}

function resetFilters() {
  period.value = "last7d";
  customStart.value = "";
  customEnd.value = "";
  selectedModels.value = [];
  sortBy.value = "token";
}

function buildQuery() {
  const query = {
    period: "",
    startAt: "",
    endAt: "",
    models: selectedModels.value.slice(),
  };
  if (period.value === "custom") {
    // 直接传本地日期（YYYY-MM-DD），由后端按本地时区闭区间处理，避免时区偏移。
    query.startAt = customStart.value;
    query.endAt = customEnd.value;
    query.period = "";
  } else {
    query.period = period.value;
  }
  return query;
}

async function loadStats() {
  await syncStats(buildQuery());
}

watch([period, customStart, customEnd, selectedModels], () => {
  void loadStats();
});

onMounted(async () => {
  await loadStats();
});
</script>

<template>
  <div class="flex h-full min-h-0 flex-col overflow-hidden pt-0 text-[#e5e5e5]">
    <div class="shrink-0 px-4 pb-4">
      <div class="flex flex-wrap items-end justify-between gap-3">
        <div class="flex flex-wrap items-end gap-3">
          <div class="flex flex-col gap-1">
            <span class="text-xs text-[#8f8f8f]">统计周期</span>
            <Select
              v-model="period"
              :options="PERIOD_OPTIONS"
              class="w-[150px]"
              aria-label="统计周期"
            />
          </div>

          <div v-if="showCustomRange" class="flex items-end gap-2">
            <div class="flex flex-col gap-1">
              <span class="text-xs text-[#8f8f8f]">起始日期</span>
              <Input v-model="customStart" type="date" class="w-[150px]" aria-label="起始日期" />
            </div>
            <div class="flex flex-col gap-1">
              <span class="text-xs text-[#8f8f8f]">结束日期</span>
              <Input v-model="customEnd" type="date" class="w-[150px]" aria-label="结束日期" />
            </div>
          </div>

          <div class="flex flex-col gap-1">
            <span class="text-xs text-[#8f8f8f]">排序</span>
            <Select
              v-model="sortBy"
              :options="SORT_OPTIONS"
              class="w-[130px]"
              aria-label="排序方式"
            />
          </div>

          <Button variant="default" @click="resetFilters">重置</Button>
        </div>
        <Button variant="default" :disabled="appState.statsLoading" @click="loadStats">刷新</Button>
      </div>

      <div class="mt-3 flex flex-wrap items-center gap-1.5">
        <span class="text-xs text-[#8f8f8f]">模型筛选：</span>
        <button
          type="button"
          class="rounded-[999px] border px-3 py-1 text-xs transition-colors duration-150"
          :class="selectedModels.length === 0
            ? 'border-[#10AD5D] bg-[#10AD5D]/15 text-[#10d06f]'
            : 'border-[#343434] bg-[#252525] text-[#a3a3a3] hover:border-[#4a4a4a] hover:text-[#e5e5e5]'"
          @click="selectedModels = []"
        >
          全部
        </button>
        <button
          v-for="option in modelOptions"
          :key="option.value"
          type="button"
          class="rounded-[999px] border px-3 py-1 text-xs transition-colors duration-150"
          :class="selectedModels.includes(option.value)
            ? 'border-[#10AD5D] bg-[#10AD5D]/15 text-[#10d06f]'
            : 'border-[#343434] bg-[#252525] text-[#a3a3a3] hover:border-[#4a4a4a] hover:text-[#e5e5e5]'"
          @click="toggleModel(option.value)"
        >
          {{ option.label }}
        </button>
        <button
          v-if="hasUnknownModel"
          type="button"
          class="rounded-[999px] border px-3 py-1 text-xs transition-colors duration-150"
          :class="selectedModels.includes('unknown')
            ? 'border-[#10AD5D] bg-[#10AD5D]/15 text-[#10d06f]'
            : 'border-[#343434] bg-[#252525] text-[#a3a3a3] hover:border-[#4a4a4a] hover:text-[#e5e5e5]'"
          @click="toggleModel('unknown')"
        >
          未知/其他
        </button>
      </div>
    </div>

    <div class="min-h-0 flex-1 overflow-y-auto scroll-shadow-bottom p-4 pt-0">
      <div v-if="appState.statsLoading" class="flex h-full min-h-[200px] items-center justify-center text-sm text-[#8f8f8f]">
        统计加载中...
      </div>

      <div v-else-if="appState.statsError" class="rounded-[8px] border border-[#4b1d1d] bg-[#2a1313] px-3 py-2 text-sm text-[#fca5a5]">
        {{ appState.statsError }}
      </div>

      <div v-else class="flex flex-col gap-4">
        <div class="grid grid-cols-4 gap-0 overflow-hidden rounded-[8px] border border-[#343434] bg-[#242424]">
          <div class="min-w-0 px-4 py-3">
            <div class="text-xs text-[#7f7f7f]">调用次数</div>
            <div class="mt-2 text-[22px] leading-none text-white" style="font-family: var(--font-num)">
              {{ formatCompactInteger(total.providerCalls) }}
            </div>
          </div>
          <div class="min-w-0 border-l border-[#343434] px-4 py-3">
            <div class="text-xs text-[#7f7f7f]">Token 总计</div>
            <div class="mt-2 text-[22px] leading-none text-white" style="font-family: var(--font-num)">
              {{ formatCompactInteger(total.totalTokens) }}
            </div>
          </div>
          <div class="min-w-0 border-l border-[#343434] px-4 py-3">
            <div class="text-xs text-[#7f7f7f]">输入 / 输出</div>
            <div class="mt-2 text-[22px] leading-none text-white" style="font-family: var(--font-num)">
              {{ formatCompactInteger(total.inputTokens) }}
              <span class="text-sm text-[#8c8c8c]">/</span>
              {{ formatCompactInteger(total.outputTokens) }}
            </div>
          </div>
          <div class="min-w-0 border-l border-[#343434] px-4 py-3">
            <div class="text-xs text-[#7f7f7f]">价格估算</div>
            <div class="mt-2 text-[22px] leading-none text-white" style="font-family: var(--font-num)">
              {{ formatUSD(totalCost) }}
            </div>
          </div>
        </div>

        <Card>
          <div class="flex items-center justify-between">
            <h3 class="text-sm font-medium text-white">按天统计</h3>
            <span class="text-xs text-[#8f8f8f]">共 {{ sortedByDay.length }} 天</span>
          </div>
          <div v-if="sortedByDay.length === 0" class="mt-4 rounded-[8px] border border-[#343434] bg-[#232323] px-3 py-6 text-center text-sm text-[#8f8f8f]">
            该时间段内暂无数据
          </div>
          <div v-else class="mt-3 overflow-x-auto">
            <table class="w-full min-w-[720px] border-collapse text-sm">
              <thead>
                <tr class="text-left text-xs text-[#8f8f8f]">
                  <th class="py-2 pr-3 font-normal">日期</th>
                  <th class="py-2 pr-3 text-right font-normal">调用次数</th>
                  <th class="py-2 pr-3 text-right font-normal">输入</th>
                  <th class="py-2 pr-3 text-right font-normal">输出</th>
                  <th class="py-2 pr-3 text-right font-normal">缓存读</th>
                  <th class="py-2 pr-3 text-right font-normal">缓存写</th>
                  <th class="py-2 pr-3 text-right font-normal">Token 总计</th>
                  <th class="py-2 pr-3 text-right font-normal">价格估算</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="bucket in sortedByDay" :key="bucket.key" class="border-t border-[#343434]">
                  <td class="py-2 pr-3 text-[#e5e5e5]">{{ bucket.label }}</td>
                  <td class="py-2 pr-3 text-right" style="font-family: var(--font-num)">{{ formatInteger(bucket.providerCalls) }}</td>
                  <td class="py-2 pr-3 text-right text-[#b9b9b9]" style="font-family: var(--font-num)">{{ formatCompactInteger(bucket.inputTokens) }}</td>
                  <td class="py-2 pr-3 text-right text-[#b9b9b9]" style="font-family: var(--font-num)">{{ formatCompactInteger(bucket.outputTokens) }}</td>
                  <td class="py-2 pr-3 text-right text-[#b9b9b9]" style="font-family: var(--font-num)">{{ formatCompactInteger(bucket.cacheReadTokens) }}</td>
                  <td class="py-2 pr-3 text-right text-[#b9b9b9]" style="font-family: var(--font-num)">{{ formatCompactInteger(bucket.cacheWriteTokens) }}</td>
                  <td class="py-2 pr-3 text-right text-white" style="font-family: var(--font-num)">{{ formatCompactInteger(bucket.totalTokens) }}</td>
                  <td class="py-2 pr-3 text-right text-[#10d06f]" style="font-family: var(--font-num)">{{ formatUSD(bucketCost(bucket)) }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </Card>

        <Card>
          <div class="flex items-center justify-between">
            <h3 class="text-sm font-medium text-white">按模型统计</h3>
            <span class="text-xs text-[#8f8f8f]">共 {{ sortedByModel.length }} 个模型</span>
          </div>
          <div v-if="sortedByModel.length === 0" class="mt-4 rounded-[8px] border border-[#343434] bg-[#232323] px-3 py-6 text-center text-sm text-[#8f8f8f]">
            该时间段内暂无数据
          </div>
          <div v-else class="mt-3 overflow-x-auto">
            <table class="w-full min-w-[560px] border-collapse text-sm">
              <thead>
                <tr class="text-left text-xs text-[#8f8f8f]">
                  <th class="py-2 pr-3 font-normal">模型</th>
                  <th class="py-2 pr-3 text-right font-normal">调用次数</th>
                  <th class="py-2 pr-3 text-right font-normal">输入</th>
                  <th class="py-2 pr-3 text-right font-normal">输出</th>
                  <th class="py-2 pr-3 text-right font-normal">Token 总计</th>
                  <th class="py-2 pr-3 text-right font-normal">价格估算</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="bucket in sortedByModel" :key="bucket.key" class="border-t border-[#343434]">
                  <td class="py-2 pr-3 text-[#e5e5e5]">{{ modelLabel(bucket.key) }}</td>
                  <td class="py-2 pr-3 text-right" style="font-family: var(--font-num)">{{ formatInteger(bucket.providerCalls) }}</td>
                  <td class="py-2 pr-3 text-right text-[#b9b9b9]" style="font-family: var(--font-num)">{{ formatCompactInteger(bucket.inputTokens) }}</td>
                  <td class="py-2 pr-3 text-right text-[#b9b9b9]" style="font-family: var(--font-num)">{{ formatCompactInteger(bucket.outputTokens) }}</td>
                  <td class="py-2 pr-3 text-right text-white" style="font-family: var(--font-num)">{{ formatCompactInteger(bucket.totalTokens) }}</td>
                  <td class="py-2 pr-3 text-right text-[#10d06f]" style="font-family: var(--font-num)">{{ formatUSD(bucketCost(bucket)) }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </Card>
      </div>
    </div>
  </div>
</template>