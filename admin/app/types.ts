export type DashboardData = {
  AdminUsername: string;
  Flash: string;
  Error: string;
  Health: HealthData;
  Stats: StatsData;
  Config: ConfigData;
  Cron: CronData;
  Usage: UsageData;
  Recent: RecentData;
};

export type HealthData = {
  DatabaseOK: boolean;
  Database: string;
  Now: string;
};

export type StatsData = {
  Users: number;
  BattleQuests: number;
  RolePlayQuests: number;
  Battles: number;
  LiveSessions: number;
  LiveStreaming: number;
  LiveEnded: number;
};

export type ConfigData = {
  AppPort: string;
  GinMode: string;
  MaxConcurrentRequests: string;
  QueueTimeoutSeconds: string;
  MaxBodyBytes: string;
  DBMaxOpenConns: string;
  DBMaxIdleConns: string;
};

export type UsageSummary = {
  CallCount: number;
  PromptTokens: number;
  CompletionTokens: number;
  TotalTokens: number;
  EstimatedCostMicros: number;
};

export type UsageRecord = {
  Id: number;
  CreatedAt: string;
  SessionMode: string;
  ProviderName: string;
  ModelName: string;
  Operation: string;
  Phase: string;
  ActorName: string;
  PromptTokens: number;
  CompletionTokens: number;
  TotalTokens: number;
  EstimatedCostMicros: number;
  BillingSource: string;
  Estimated: boolean;
};

export type UsageData = {
  Total: UsageSummary;
  Battle: UsageSummary;
  RolePlay: UsageSummary;
  Recent: UsageRecord[];
  PricingHint: string;
};

export type CronJobData = {
  LastRunID: string;
  LastProvider: string;
  LastModel: string;
  LastStep: string;
  LastStatus: string;
  LastDurationMS: number;
  LastMessage: string;
  LastError: string;
};

export type CronLogData = {
  CreatedAt: string;
  Job: string;
  RunID: string;
  Provider: string;
  Model: string;
  Step: string;
  Status: string;
  Message: string;
};

export type CronData = {
  Enabled: boolean;
  Timezone: string;
  Window: string;
  Limit: number;
  NextRun: string;
  Battle: CronJobData;
  RolePlay: CronJobData;
  Logs: CronLogData[];
};

export type BattleQuest = {
  Id: number;
  CreatedAt: string;
  Title: string;
  Content: string;
  Level: string;
  Point: number;
  Theme: string;
  Xp: number;
  Coin: number;
  Status: string;
};

export type RolePlayQuest = {
  Id: number;
  CreatedAt: string;
  Title: string;
  Summary: string;
  Prompt: string;
  Theme: string;
  Level: string;
  Xp: number;
  Coin: number;
  Status: string;
};

export type Battle = {
  Id: number;
  UpdatedAt: string;
  Title: string;
  Status: string;
  CurrentRound: number;
  TotalRounds: number;
  TotalTokens: number;
};

export type LiveSession = {
  Id: number;
  CreatedAt: string;
  UpdatedAt: string;
  ChannelKey: string;
  Mode: string;
  Status: string;
  ViewerCount: number;
  AllowReplay: boolean;
};

export type RecentData = {
  BattleQuests: BattleQuest[];
  RolePlayQuests: RolePlayQuest[];
  Battles: Battle[];
  LiveSessions: LiveSession[];
};

export type AccountSummary = {
  totalAccounts: number;
  updatedLast7Days: number;
  updatedLast30Days: number;
  totalXp: number;
  totalCoins: number;
};

export type Account = {
  id: number;
  createdAt: string;
  updatedAt: string;
  pseudo: string;
  email: string;
  avatar: string;
  xp: number;
  coin: number;
  battleCount: number;
  rolePlayCount: number;
  iaProfileCount: number;
  liveSessionCount: number;
};

export type AccountsResponse = {
  summary: AccountSummary;
  accounts: Account[];
};

export type SystemResponse = {
  health: HealthData;
  config: ConfigData;
  runtime: {
    startedAt: string;
    uptimeSeconds: number;
    goVersion: string;
    goos: string;
    goarch: string;
    numCpu: number;
    numGoroutine: number;
    allocBytes: number;
    heapAllocBytes: number;
    sysBytes: number;
    numGc: number;
  };
  requests: {
    totalRequests: number;
    activeRequests: number;
    status2xx: number;
    status3xx: number;
    status4xx: number;
    status5xx: number;
    averageLatencyMs: number;
    maxLatencyMs: number;
  };
  database: {
    maxOpenConnections: number;
    openConnections: number;
    inUse: number;
    idle: number;
    waitCount: number;
    waitDuration: number;
    maxIdleClosed: number;
    maxIdleTimeClosed: number;
    maxLifetimeClosed: number;
  };
  network: {
    liveSessions: number;
    liveStreaming: number;
    liveEnded: number;
    liveViewers: number;
    arenas: number;
    coopParties: number;
  };
};

export type NexusCoinStats = {
  callCount: number;
  totalTokens: number;
  totalCostMicros: number;
  averageTokensPerCall: number;
  averageCostMicrosPerToken: number;
  marginPercent: number;
  costSource: string;
};

export type NexusCoinPlan = {
  id: number;
  slug: string;
  position: number;
  name: string;
  subtitle: string;
  description: string;
  status: string;
  tokenBudget: number;
  nexusCoins: number;
  baseCostMicros: number;
  marginPercent: number;
  priceMicros: number;
  estimatedCalls: number;
  estimatedTokensPerCall: number;
};

export type NexusCoinEstimate = Omit<NexusCoinPlan, "id" | "position" | "status"> & {
  costSource: string;
};

export type NexusCoinResponse = {
  stats: NexusCoinStats;
  estimations: NexusCoinEstimate[];
  plans: NexusCoinPlan[];
};

export type AdminRolePlayChapter = {
  id: number;
  position: number;
  title: string;
  summary: string;
  objective: string;
  isBoss: boolean;
  xp: number;
  coin: number;
};

export type AdminRolePlayArc = {
  id: number;
  position: number;
  title: string;
  summary: string;
  objective: string;
  chapterCount: number;
  chapters: AdminRolePlayChapter[];
};

export type AdminRolePlayQuest = {
  id: number;
  createdAt: string;
  updatedAt: string;
  slug: string;
  title: string;
  summary: string;
  prompt: string;
  theme: string;
  level: string;
  xp: number;
  coin: number;
  source: string;
  status: string;
  arcCount: number;
  chapterCount: number;
  arcs: AdminRolePlayArc[];
};

export type AdminRolePlayQuestsResponse = {
  stats: {
    totalQuests: number;
    published: number;
    draft: number;
    archived: number;
    totalArcs: number;
    totalChapters: number;
  };
  quests: AdminRolePlayQuest[];
};
