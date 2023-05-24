package exporter

import (
	"context"
	"strconv"
	"sync/atomic"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/savingsplans"
	savingsplansTypes "github.com/aws/aws-sdk-go-v2/service/savingsplans/types"
	log "github.com/sirupsen/logrus"
)

type savingPlanProperties struct {
	Region             string
	InstanceType       string
	InstanceFamily     string
	ProductDescription string
	Tenancy            string
}

func (e *Exporter) getSavingPlanPricing(region string, scrapes chan<- scrapeResult) {
	tmpCfg := e.awsCfg

	client := savingsplans.NewFromConfig(tmpCfg)

	params := &savingsplans.DescribeSavingsPlansOfferingRatesInput{
		MaxResults:       *aws.Int32((AwsMaxResultsPerPage)),
		SavingsPlanTypes: convertSavingsPlanType(e.savingPlanTypes),
		ServiceCodes:     []savingsplansTypes.SavingsPlanRateServiceCode{"AmazonEC2"},
		Filters: []savingsplansTypes.SavingsPlanOfferingRateFilterElement{
			{
				Name:   savingsplansTypes.SavingsPlanRateFilterAttributeRegion,
				Values: []string{region},
			},
			{
				Name:   savingsplansTypes.SavingsPlanRateFilterAttributeTenancy,
				Values: []string{"shared"},
			},
			{
				Name:   savingsplansTypes.SavingsPlanRateFilterAttributeProductDescription,
				Values: e.productDescriptions,
			},
		},
	}

	savingPlanList := make([]savingsplansTypes.SavingsPlanOfferingRate, 0)

	for {
		resp, err := client.DescribeSavingsPlansOfferingRates(context.TODO(), params)

		if err != nil {
			log.WithError(err).Errorf("error while fetching saving plans [region=%s]", region)
			atomic.AddUint64(&e.errorCount, 1)
		}

		savingPlanList = append(savingPlanList, resp.SearchResults...)

		if *resp.NextToken == "" {
			break
		}

		params.NextToken = resp.NextToken
	}

	for _, plan := range savingPlanList {
		planProperties := convertPropertiesToStruct(plan.Properties)

		if !isMatchAny(e.instanceRegexes, planProperties.InstanceType) {
			log.Debugf("Skipping instance type: %s", planProperties.InstanceType)
			continue
		}

		value, err := strconv.ParseFloat(*plan.Rate, 64)
		if err != nil {
			log.WithError(err).Errorf("error while parsing saving plan price value from API response [region=%s, type=%s]", region, planProperties.InstanceType)
			atomic.AddUint64(&e.errorCount, 1)
		}
		log.Debugf("Creating new metric: ec2{region=%s, instance_type=%s, product_description=%s} = %v.", region, planProperties.InstanceType, planProperties.ProductDescription, value)

		vcpu, memory := e.getNormalizedCost(value, planProperties.InstanceType)
		scrapes <- scrapeResult{
			Name:               "ec2",
			Value:              value,
			Region:             region,
			InstanceType:       planProperties.InstanceType,
			InstanceLifecycle:  "ondemand",
			ProductDescription: planProperties.ProductDescription,
			SavingPlanOption:   string(plan.SavingsPlanOffering.PaymentOption),
			SavingPlanDuration: SecondsToYears(plan.SavingsPlanOffering.DurationSeconds),
			SavingPlanType:     string(plan.SavingsPlanOffering.PlanType),
			Memory:             e.getInstanceMemory(planProperties.InstanceType),
			VCpu:               e.getInstanceVCpu(planProperties.InstanceType),
		}
		scrapes <- scrapeResult{
			Name:               "ec2_memory",
			Value:              memory,
			Region:             region,
			InstanceType:       planProperties.InstanceType,
			InstanceLifecycle:  "ondemand",
			SavingPlanOption:   string(plan.SavingsPlanOffering.PaymentOption),
			SavingPlanDuration: SecondsToYears(plan.SavingsPlanOffering.DurationSeconds),
			SavingPlanType:     string(plan.SavingsPlanOffering.PlanType),
		}
		scrapes <- scrapeResult{
			Name:               "ec2_vcpu",
			Value:              vcpu,
			Region:             region,
			InstanceType:       planProperties.InstanceType,
			InstanceLifecycle:  "ondemand",
			SavingPlanOption:   string(plan.SavingsPlanOffering.PaymentOption),
			SavingPlanDuration: SecondsToYears(plan.SavingsPlanOffering.DurationSeconds),
			SavingPlanType:     string(plan.SavingsPlanOffering.PlanType),
		}
	}
}

func convertSavingsPlanType(spt []string) []savingsplansTypes.SavingsPlanType {
	result := make([]savingsplansTypes.SavingsPlanType, 0)

	for _, v := range spt {
		result = append(result, savingsplansTypes.SavingsPlanType(v))
	}

	return result
}

func convertPropertiesToStruct(properties []savingsplansTypes.SavingsPlanOfferingRateProperty) savingPlanProperties {
	result := savingPlanProperties{}

	for _, property := range properties {
		if property.Name != nil && property.Value != nil {
			switch *property.Name {
			case string(savingsplansTypes.SavingsPlanRatePropertyKeyRegion):
				result.Region = *property.Value
			case string(savingsplansTypes.SavingsPlanRatePropertyKeyInstanceType):
				result.InstanceType = *property.Value
			case string(savingsplansTypes.SavingsPlanRatePropertyKeyInstanceFamily):
				result.InstanceFamily = *property.Value
			case string(savingsplansTypes.SavingsPlanRatePropertyKeyProductDescription):
				result.ProductDescription = *property.Value
			case string(savingsplansTypes.SavingsPlanRatePropertyKeyTenancy):
				result.Tenancy = *property.Value
			}
		}
	}

	return result
}

func SecondsToYears(seconds int64) int {
	const secondsPerYear = 31536000 // seconds in 1 year

	years := seconds / secondsPerYear

	if years != 1 && years != 3 {
		panic("Value could be only 1 or 3.")
	}

	return int(years)
}
