package dynamicmodule

import systemi18n "pantheon-ops/backend/modules/system/i18n"

type ModuleI18nLifecycleSummary struct {
	Module                         string   `json:"module"`
	Triggered                      bool     `json:"triggered"`
	ObservedKeys                   []string `json:"observedKeys"`
	ObservedRows                   int64    `json:"observedRows"`
	ArchivedKeys                   []string `json:"archivedKeys"`
	ArchivedRows                   int64    `json:"archivedRows"`
	DeletedKeys                    []string `json:"deletedKeys"`
	DeletedRows                    int64    `json:"deletedRows"`
	ObservationOnly                bool     `json:"observationOnly"`
	ArchivedRetentionThresholdDays int64    `json:"archivedRetentionThresholdDays"`
}

func buildModuleI18nLifecycleSummary(moduleName string, triggered bool, resp *systemi18n.I18nUnusedLifecycleAdvanceResp) *ModuleI18nLifecycleSummary {
	summary := &ModuleI18nLifecycleSummary{
		Module:                         moduleName,
		Triggered:                      triggered,
		ObservedKeys:                   make([]string, 0),
		ArchivedKeys:                   make([]string, 0),
		DeletedKeys:                    make([]string, 0),
		ArchivedRetentionThresholdDays: systemi18n.I18nArchivedRetentionThresholdDays,
	}
	if resp == nil {
		return summary
	}
	if resp.Module != "" {
		summary.Module = resp.Module
	}
	summary.ObservedKeys = append(summary.ObservedKeys, resp.ObservedKeys...)
	summary.ObservedRows = resp.ObservedRows
	summary.ArchivedKeys = append(summary.ArchivedKeys, resp.ArchivedKeys...)
	summary.ArchivedRows = resp.ArchivedRows
	summary.DeletedKeys = append(summary.DeletedKeys, resp.DeletedKeys...)
	summary.DeletedRows = resp.DeletedRows
	summary.ObservationOnly = resp.ObservationOnly
	if resp.ArchivedRetentionThresholdDays > 0 {
		summary.ArchivedRetentionThresholdDays = resp.ArchivedRetentionThresholdDays
	}
	return summary
}
