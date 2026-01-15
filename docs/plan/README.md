# SCRM 插件模块计划总览

## 1. 文档目的与产出物
- 基于 PowerXDocs 的场景化用例，形成 SCRM 插件模块化规划与索引。
- 建立 `docs/plan/<module>` 的规划入口，便于持续补齐 PRD、接口与实施节奏。
- 为跨团队协作提供统一视图，减少重复沟通与范围漂移。

## 2. 模块索引
| 模块 | 目录 | 主用例数量 |
| --- | --- | --- |
| AIGC 自动化智能 | `docs/plan/aigc_automation_intelligence` | 4 |
| 数据分析与洞察 | `docs/plan/analytics_insights` | 2 |
| 社群与客户运营 | `docs/plan/community_customer_engagement` | 2 |
| 合规、安全与风控 | `docs/plan/compliance_security_risk_control` | 2 |
| 内容分发与互动自动化 | `docs/plan/content_engagement_automation` | 2 |
| 客户成功与运营协作闭环 | `docs/plan/customer_service_collaboration_loop` | 2 |
| 线索获取与智能分配 | `docs/plan/lead_capture_smart_assignment` | 2 |
| 移动前线作业能力 | `docs/plan/mobile_frontline_capabilities` | 4 |
| 平台生态与可扩展性 | `docs/plan/platform_ecosystem_extensibility` | 4 |
| 智能标签与客户分群 | `docs/plan/smart_tagging_customer_segmentation` | 4 |
| 社交触点接入与账号治理 | `docs/plan/social_channel_governance` | 2 |
| 社交交易与分销 | `docs/plan/social_commerce_distribution` | 4 |
| 社交销售与外勤协同 | `docs/plan/social_selling_field_collab` | 2 |
| 系统集成与数据流转 | `docs/plan/system_integration_data_orchestration` | 4 |

## 3. 排序与依赖划分
### 3.1 推荐推进顺序
1. 社交触点接入与账号治理
2. 线索获取与智能分配
3. 社群与客户运营
4. 内容分发与互动自动化
5. 社交销售与外勤协同
6. 移动前线作业能力
7. 智能标签与客户分群
8. 客户成功与运营协作闭环
9. 系统集成与数据流转
10. 社交交易与分销
11. 合规、安全与风控
12. 数据分析与洞察
13. AIGC 自动化智能
14. 平台生态与可扩展性

### 3.2 依赖边界说明
- **SCRM 自身闭环优先**：社交触点接入与账号治理、线索获取与智能分配、社群与客户运营、内容分发与互动自动化、社交销售与外勤协同、移动前线作业能力、智能标签与客户分群、客户成功与运营协作闭环。
- **需对接 PowerX 底座**：系统集成与数据流转、社交交易与分销、合规、安全与风控、数据分析与洞察、AIGC 自动化智能、平台生态与可扩展性。

### 3.3 Skeleton 能力封装与多实现策略
- **统一对外接口**：即使在 Skeleton 模式下，也必须暴露与 PowerX 底座一致的能力契约与调用接口，保证前端/工作流/集成侧的调用方式不变。
- **实现可切换**：能力实现可在 Skeleton 与 PowerX 底座之间替换，遵循“接口封装 + 运行态选择”的原则，避免业务方感知底层差异。
- **参考规范**：
  - `PowerXDocs/docs/standards/powerx/backend/integration/02_capability/Capability_Contract_Spec.md`
  - `PowerXDocs/docs/standards/powerx/backend/integration/02_capability/Transport_Adapter_Spec.md`
  - `PowerXDocs/docs/standards/powerx/backend/integration/06_gateway/Integration_API_and_Admin_Interface.md`
  - `PowerXDocs/docs/guides/publish/local-dev-debug.md`（Skeleton 与宿主保持一致 API 契约）

## 4. 使用方式
1. 先在本 README 确认模块入口与主用例数量。
2. 进入对应模块目录，阅读场景清单与规划要点。
3. 需要落地细节时，补充对应模块 README 的子章节或新增子文档。

## 5. 迭代建议
- 以“社交渠道治理 → 线索与客户运营 → 内容与交易 → 数据与智能”的顺序推进。
- 每个模块先完成“核心流程 + 关键数据回流 + 权限审计”。
- 模块达成后再扩展自动化、智能化与生态能力。

## 6. 下一步动作
- [ ] 对每个模块补充具体页面/接口映射（如需）。
- [ ] 明确依赖系统与数据来源，建立联调清单。
- [ ] 按季度维护优先级与里程碑。
