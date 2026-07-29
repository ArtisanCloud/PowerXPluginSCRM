package seed

import (
	"context"
	"errors"
	"strings"
	"time"

	oppmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/domain/models/opportunity"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type opportunityPipelineTemplateSeed struct {
	Key         string
	GroupKey    string
	Name        string
	Label       string
	Segment     string
	Description string
	SortOrder   int
	Stages      []opportunityPipelineTemplateStageSeed
}

type opportunityPipelineTemplateStageSeed struct {
	Key      string
	Label    string
	Order    int
	Rate     int
	SLA      int
	Type     string
	Fixed    string
	IsActive bool
}

func seedOpportunityPipelineTemplates(ctx context.Context, db *gorm.DB) error {
	if db == nil {
		return errors.New("seed opportunity pipeline templates requires database")
	}
	ctxDB := db.WithContext(ctx)
	if !ctxDB.Migrator().HasTable(&oppmodel.OpportunityPipelineTemplate{}) {
		return errors.New("seed opportunity pipeline templates requires template table")
	}
	if !ctxDB.Migrator().HasTable(&oppmodel.OpportunityPipelineTemplateStage{}) {
		return errors.New("seed opportunity pipeline templates requires template stage table")
	}
	now := time.Now().UTC()
	for _, item := range defaultOpportunityPipelineTemplates() {
		templateKey := strings.TrimSpace(strings.ToLower(item.Key))
		if templateKey == "" {
			continue
		}
		template := oppmodel.OpportunityPipelineTemplate{
			TemplateUUID: uuid.NewString(),
			TemplateKey:  templateKey,
			GroupKey:     strings.TrimSpace(strings.ToLower(item.GroupKey)),
			Name:         strings.TrimSpace(item.Name),
			Label:        strings.TrimSpace(item.Label),
			Segment:      strings.TrimSpace(item.Segment),
			Description:  strings.TrimSpace(item.Description),
			SortOrder:    item.SortOrder,
			IsActive:     true,
			CreatedAt:    now,
			UpdatedAt:    now,
		}
		if err := ctxDB.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "template_key"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"group_key", "name", "label", "segment", "description", "sort_order", "is_active", "updated_at",
			}),
		}).Create(&template).Error; err != nil {
			return err
		}
		if err := ctxDB.Where("template_key = ?", templateKey).
			Delete(&oppmodel.OpportunityPipelineTemplateStage{}).Error; err != nil {
			return err
		}
		stages := make([]oppmodel.OpportunityPipelineTemplateStage, 0, len(item.Stages))
		for _, stage := range item.Stages {
			isActive := stage.IsActive
			if !isActive {
				isActive = true
			}
			stages = append(stages, oppmodel.OpportunityPipelineTemplateStage{
				StageUUID:      uuid.NewString(),
				TemplateKey:    templateKey,
				StageKey:       strings.TrimSpace(strings.ToLower(stage.Key)),
				Label:          strings.TrimSpace(stage.Label),
				SortOrder:      stage.Order,
				DefaultWinRate: stage.Rate,
				SLADays:        stage.SLA,
				StageType:      strings.TrimSpace(strings.ToLower(stage.Type)),
				FixedStage:     strings.TrimSpace(strings.ToLower(stage.Fixed)),
				IsActive:       isActive,
				CreatedAt:      now,
				UpdatedAt:      now,
			})
		}
		if len(stages) > 0 {
			if err := ctxDB.Create(&stages).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

func defaultOpportunityPipelineTemplates() []opportunityPipelineTemplateSeed {
	stage := func(key, label string, order, rate, sla int, stageType, fixed string) opportunityPipelineTemplateStageSeed {
		return opportunityPipelineTemplateStageSeed{Key: key, Label: label, Order: order, Rate: rate, SLA: sla, Type: stageType, Fixed: fixed, IsActive: true}
	}
	defaultTerminal := []opportunityPipelineTemplateStageSeed{
		stage(oppmodel.StageWon, "赢单", 90, 100, 0, oppmodel.StageTypeWon, oppmodel.StageWon),
		stage(oppmodel.StageLost, "输单", 100, 0, 0, oppmodel.StageTypeLost, oppmodel.StageLost),
	}
	withTerminal := func(items ...opportunityPipelineTemplateStageSeed) []opportunityPipelineTemplateStageSeed {
		out := append([]opportunityPipelineTemplateStageSeed{}, items...)
		out = append(out, defaultTerminal...)
		return out
	}
	return []opportunityPipelineTemplateSeed{
		{
			Key: "b2b_standard", GroupKey: "b2b-standard", Name: "通用 B2B 销售流程", Label: "通用 B2B 销售", Segment: "通用", SortOrder: 10,
			Description: "适合标准线索确认、方案、谈判、赢输单的基础商机漏斗。",
			Stages: append([]opportunityPipelineTemplateStageSeed{
				stage(oppmodel.StageOpen, "打开", 10, 20, 3, oppmodel.StageTypeActive, oppmodel.StageOpen),
				stage(oppmodel.StageQualified, "已确认", 20, 40, 5, oppmodel.StageTypeActive, oppmodel.StageQualified),
				stage(oppmodel.StageProposal, "方案", 30, 60, 7, oppmodel.StageTypeActive, oppmodel.StageProposal),
				stage(oppmodel.StageNegotiation, "谈判", 40, 80, 10, oppmodel.StageTypeActive, oppmodel.StageNegotiation),
			}, defaultTerminal...),
		},
		{
			Key: "enterprise_account", GroupKey: "enterprise-account", Name: "企业大客户商机模型", Label: "企业大客户销售", Segment: "B2B", SortOrder: 20,
			Description: "适合长周期、多角色决策的大客户销售，覆盖调研、方案共创、商业论证、采购法务和合同定稿。",
			Stages: withTerminal(
				stage("identified", "目标客户识别", 10, 10, 7, oppmodel.StageTypeActive, oppmodel.StageOpen),
				stage("discovery", "需求调研", 20, 25, 10, oppmodel.StageTypeActive, oppmodel.StageQualified),
				stage("solution", "方案共创", 30, 45, 14, oppmodel.StageTypeActive, oppmodel.StageProposal),
				stage("business_case", "商业论证", 40, 60, 14, oppmodel.StageTypeActive, oppmodel.StageProposal),
				stage("procurement", "采购与法务", 50, 75, 21, oppmodel.StageTypeActive, oppmodel.StageNegotiation),
				stage("contracting", "合同定稿", 60, 90, 14, oppmodel.StageTypeActive, oppmodel.StageNegotiation),
			),
		},
		{
			Key: "b2b_industrial", GroupKey: "b2b-industrial", Name: "2B 工业项目销售模型", Label: "2B 工业项目销售", Segment: "B2B", SortOrder: 30,
			Description: "适合工业设备、自动化、工程项目等长周期销售，覆盖立项、技术方案、样机验证、商务报价和验收交付。",
			Stages: withTerminal(
				stage("project_intake", "项目线索接入", 10, 15, 5, oppmodel.StageTypeActive, oppmodel.StageOpen),
				stage("site_survey", "现场调研/工况确认", 20, 30, 14, oppmodel.StageTypeActive, oppmodel.StageQualified),
				stage("technical_solution", "技术方案设计", 30, 45, 21, oppmodel.StageTypeActive, oppmodel.StageProposal),
				stage("pilot_validation", "样机/试点验证", 40, 60, 30, oppmodel.StageTypeActive, oppmodel.StageProposal),
				stage("commercial_quote", "商务报价", 50, 70, 14, oppmodel.StageTypeActive, oppmodel.StageNegotiation),
				stage("acceptance_contract", "验收条款与合同", 60, 88, 21, oppmodel.StageTypeActive, oppmodel.StageNegotiation),
			),
		},
		{
			Key: "automotive_manufacturing", GroupKey: "automotive-manufacturing", Name: "汽车制造客户商机模型", Label: "汽车制造业", Segment: "B2B", SortOrder: 40,
			Description: "适合主机厂/零部件客户，从 RFQ、技术评审、样件试制、成本报价到供应商定点和 SOP 交付。",
			Stages: withTerminal(
				stage("rfq", "RFQ/询价接入", 10, 15, 5, oppmodel.StageTypeActive, oppmodel.StageOpen),
				stage("technical_review", "技术评审", 20, 30, 14, oppmodel.StageTypeActive, oppmodel.StageQualified),
				stage("sample_trial", "样件/试制验证", 30, 45, 30, oppmodel.StageTypeActive, oppmodel.StageProposal),
				stage("costing", "成本核算与报价", 40, 60, 14, oppmodel.StageTypeActive, oppmodel.StageProposal),
				stage("supplier_nomination", "供应商定点", 50, 80, 30, oppmodel.StageTypeActive, oppmodel.StageNegotiation),
				stage("sop_contract", "SOP/合同交付", 60, 90, 30, oppmodel.StageTypeActive, oppmodel.StageNegotiation),
			),
		},
		{
			Key: "healthcare", GroupKey: "healthcare", Name: "医疗健康商机模型", Label: "医疗健康", Segment: "行业", SortOrder: 50,
			Description: "适合医院、科室、器械或解决方案销售，覆盖临床演示、预算立项、招采投标和合规商务评审。",
			Stages: withTerminal(
				stage("department_need", "科室需求确认", 10, 15, 7, oppmodel.StageTypeActive, oppmodel.StageOpen),
				stage("clinical_demo", "临床演示/试用", 20, 35, 21, oppmodel.StageTypeActive, oppmodel.StageQualified),
				stage("budget_approval", "预算立项", 30, 50, 30, oppmodel.StageTypeActive, oppmodel.StageProposal),
				stage("tender", "招采/投标", 40, 65, 30, oppmodel.StageTypeActive, oppmodel.StageProposal),
				stage("compliance_review", "合规与商务评审", 50, 80, 21, oppmodel.StageTypeActive, oppmodel.StageNegotiation),
				stage("contract_delivery", "合同与交付", 60, 90, 14, oppmodel.StageTypeActive, oppmodel.StageNegotiation),
			),
		},
		{
			Key: "b2c_food_service", GroupKey: "b2c-food-service", Name: "2C 餐饮客户转化模型", Label: "2C 餐饮转化", Segment: "2C", SortOrder: 60,
			Description: "适合餐饮门店私域转化，从触达领券、预约到店、点单消费、复购唤醒到会员沉淀。",
			Stages: withTerminal(
				stage("first_touch", "触达/领券", 10, 15, 1, oppmodel.StageTypeActive, oppmodel.StageOpen),
				stage("intent_confirmed", "意向确认", 20, 30, 2, oppmodel.StageTypeActive, oppmodel.StageQualified),
				stage("reservation", "预约到店", 30, 45, 3, oppmodel.StageTypeActive, oppmodel.StageProposal),
				stage("visit_order", "到店点单", 40, 65, 1, oppmodel.StageTypeActive, oppmodel.StageProposal),
				stage("upsell_member", "加购/会员转化", 50, 80, 7, oppmodel.StageTypeActive, oppmodel.StageNegotiation),
				stage("repurchase", "复购唤醒", 60, 90, 14, oppmodel.StageTypeActive, oppmodel.StageNegotiation),
			),
		},
		{
			Key: "b2c_health_service", GroupKey: "b2c-health-service", Name: "2C 健康服务转化模型", Label: "2C 健康服务", Segment: "2C", SortOrder: 70,
			Description: "适合体检、医美、康复、健康管理等服务，从咨询评估、方案建议、预约服务到复诊续费。",
			Stages: withTerminal(
				stage("consultation", "咨询建档", 10, 15, 1, oppmodel.StageTypeActive, oppmodel.StageOpen),
				stage("assessment", "健康评估", 20, 30, 3, oppmodel.StageTypeActive, oppmodel.StageQualified),
				stage("service_plan", "方案建议", 30, 50, 5, oppmodel.StageTypeActive, oppmodel.StageProposal),
				stage("appointment", "预约服务", 40, 65, 7, oppmodel.StageTypeActive, oppmodel.StageProposal),
				stage("service_visit", "到诊/到店服务", 50, 80, 7, oppmodel.StageTypeActive, oppmodel.StageNegotiation),
				stage("renewal_followup", "复诊/续费跟进", 60, 90, 14, oppmodel.StageTypeActive, oppmodel.StageNegotiation),
			),
		},
		{
			Key: "financial_services", GroupKey: "financial-services", Name: "金融服务商机模型", Label: "金融服务", Segment: "行业", SortOrder: 80,
			Description: "适合金融产品或机构客户销售，覆盖客户准入、风险评估、产品匹配、审批评审、KYC 和开通。",
			Stages: withTerminal(
				stage("client_profile", "客户画像与准入", 10, 20, 5, oppmodel.StageTypeActive, oppmodel.StageOpen),
				stage("risk_assessment", "风险评估", 20, 35, 10, oppmodel.StageTypeActive, oppmodel.StageQualified),
				stage("solution_match", "产品方案匹配", 30, 55, 10, oppmodel.StageTypeActive, oppmodel.StageProposal),
				stage("committee_review", "审批/委员会评审", 40, 70, 14, oppmodel.StageTypeActive, oppmodel.StageNegotiation),
				stage("contract_kyc", "合同与KYC", 50, 85, 14, oppmodel.StageTypeActive, oppmodel.StageNegotiation),
				stage("funding_activation", "放款/开通", 60, 95, 7, oppmodel.StageTypeActive, oppmodel.StageNegotiation),
			),
		},
	}
}
