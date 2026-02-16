import React, { useEffect, useState } from 'react';
import { Layers, Save, Check, X, Plus, Trash2, Settings } from 'lucide-react';
import api from '../services/api';
import { Card, CardContent, CardHeader, CardTitle } from '../components/ui/Card';
import { Button } from '../components/ui/Button';
import { Input } from '../components/ui/Input';
import { useToast } from '../components/ui/Toast';
import WorkflowEditor from '../components/WorkflowEditor';
import type { LeaveType, LeaveWorkflow } from '../types';

interface LeaveTypeConfig {
    id: string;
    leave_type: string;
    base_entitlement: number;
    years_of_service_tiers: Record<string, number> | null;
    prorate_first_year: boolean;
    allow_carry_forward: boolean;
    max_carry_forward_days: number;
    max_days_per_application: number | null;
    requires_attachment: boolean;
    min_advance_days: number;
    is_active: boolean;
    display_order: number;
}

interface ServiceTier {
    years: string;
    bonus: number;
}

const leaveTypeLabels: Record<string, string> = {
    'annual': 'Annual Leave',
    'sick': 'Sick Leave',
    'maternity': 'Maternity Leave',
    'paternity': 'Paternity Leave',
    'emergency': 'Emergency Leave',
    'unpaid': 'Unpaid Leave',
    'unrecorded': 'Unrecorded Leave',
    'hospitalization': 'Hospitalization Leave',
};

const LeaveTypeSettings: React.FC = () => {
    const [loading, setLoading] = useState(true);
    const [saving, setSaving] = useState<string | null>(null);
    const [configs, setConfigs] = useState<LeaveTypeConfig[]>([]);
    const [workflows, setWorkflows] = useState<Record<string, LeaveWorkflow>>({});
    const [editingId, setEditingId] = useState<string | null>(null);
    const [editForm, setEditForm] = useState<Partial<LeaveTypeConfig>>({});
    const [serviceTiers, setServiceTiers] = useState<ServiceTier[]>([]);
    const { showToast } = useToast();
    const [workflowLeaveType, setWorkflowLeaveType] = useState<LeaveType | null>(null);

    const fetchData = async () => {
        setLoading(true);
        try {
            const [configsRes, workflowsRes] = await Promise.all([
                api.get('/admin/leave-type-configs'),
                api.get<LeaveWorkflow[]>('/admin/workflows')
            ]);
            setConfigs(configsRes.data);

            // Map workflows by leave type for easy lookup
            const workflowMap: Record<string, LeaveWorkflow> = {};
            workflowsRes.data.forEach(wf => {
                // Only store active workflows or the latest version if multiple
                if (wf.is_active) {
                    workflowMap[wf.leave_type] = wf;
                }
            });
            setWorkflows(workflowMap);
        } catch (err) {
            console.error("Failed to fetch data", err);
            showToast("Failed to load settings", "error");
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => {
        fetchData();
    }, []);

    const handleEdit = (config: LeaveTypeConfig) => {
        setEditingId(config.id);
        setEditForm({
            base_entitlement: config.base_entitlement,
            prorate_first_year: config.prorate_first_year,
            allow_carry_forward: config.allow_carry_forward,
            max_carry_forward_days: config.max_carry_forward_days,
            requires_attachment: config.requires_attachment,
            min_advance_days: config.min_advance_days,
            is_active: config.is_active,
        });

        // Convert tiers object to array for editing
        const tiers: ServiceTier[] = [];
        if (config.years_of_service_tiers) {
            Object.entries(config.years_of_service_tiers).forEach(([years, bonus]) => {
                tiers.push({ years, bonus: Number(bonus) });
            });
        }
        // Sort by years
        tiers.sort((a, b) => parseInt(a.years) - parseInt(b.years));
        setServiceTiers(tiers);
    };

    const handleCancel = () => {
        setEditingId(null);
        setEditForm({});
        setServiceTiers([]);
    };

    const handleAddTier = () => {
        setServiceTiers([...serviceTiers, { years: '', bonus: 0 }]);
    };

    const handleRemoveTier = (index: number) => {
        setServiceTiers(serviceTiers.filter((_, i) => i !== index));
    };

    const handleTierChange = (index: number, field: 'years' | 'bonus', value: string | number) => {
        const updated = [...serviceTiers];
        if (field === 'years') {
            updated[index].years = String(value);
        } else {
            updated[index].bonus = Number(value);
        }
        setServiceTiers(updated);
    };

    const handleSave = async (leaveType: string) => {
        setSaving(leaveType);

        // Convert tiers array back to object
        const tiersObject: Record<string, number> = {};
        serviceTiers.forEach(tier => {
            if (tier.years && tier.years.trim() !== '') {
                tiersObject[tier.years] = tier.bonus;
            }
        });

        const payload = {
            ...editForm,
            years_of_service_tiers: Object.keys(tiersObject).length > 0 ? tiersObject : null,
        };

        try {
            await api.put(`/admin/leave-type-configs/${leaveType}`, payload);
            showToast(`${leaveTypeLabels[leaveType] || leaveType} configuration updated successfully`, "success");
            setEditingId(null);
            setEditForm({});
            setServiceTiers([]);
            fetchData();
        } catch (err: any) {
            showToast(err.response?.data?.error || "Failed to update configuration", "error");
        } finally {
            setSaving(null);
        }
    };

    if (loading) {
        return (
            <div className="flex justify-center items-center h-64">
                <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-brand-600"></div>
            </div>
        );
    }

    return (
        <div className="space-y-6">
            <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4">
                <div>
                    <h1 className="text-2xl font-bold text-slate-900">Leave Type Settings</h1>
                    <p className="text-slate-500 mt-1">Configure entitlements for each leave type</p>
                </div>
            </div>

            <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
                {configs.map((config) => (
                    <Card key={config.id} className={!config.is_active ? 'opacity-60' : ''}>
                        <CardHeader>
                            <CardTitle className="flex items-center justify-between">
                                <div className="flex items-center gap-2">
                                    <Layers className="h-5 w-5 text-brand-600" />
                                    {leaveTypeLabels[config.leave_type] || config.leave_type}
                                </div>
                                {!config.is_active && (
                                    <span className="text-xs bg-slate-200 text-slate-600 px-2 py-1 rounded">
                                        Inactive
                                    </span>
                                )}
                            </CardTitle>
                        </CardHeader>
                        <CardContent>
                            <div className="mb-4 flex justify-end">
                                <Button
                                    variant="outline"
                                    onClick={() => setWorkflowLeaveType(config.leave_type as LeaveType)}
                                    className="text-sm flex items-center gap-1"
                                >
                                    <Settings className="h-4 w-4" />
                                    Workflow
                                    {workflows[config.leave_type] && (
                                        <span className="ml-1 px-1.5 py-0.5 bg-brand-100 text-brand-700 rounded text-xs font-medium">
                                            v{workflows[config.leave_type].version}
                                        </span>
                                    )}
                                </Button>
                            </div>
                            {editingId === config.id ? (
                                <div className="space-y-4">
                                    <div className="space-y-1">
                                        <label className="text-sm font-medium text-slate-700">
                                            Base Entitlement (days)
                                        </label>
                                        <Input
                                            type="number"
                                            value={editForm.base_entitlement ?? 0}
                                            onChange={(e) => setEditForm({
                                                ...editForm,
                                                base_entitlement: parseFloat(e.target.value) || 0
                                            })}
                                        />
                                    </div>

                                    {/* Years of Service Bonus Editor */}
                                    <div className="space-y-3">
                                        <div className="flex justify-between items-center">
                                            <label className="text-sm font-medium text-slate-700">
                                                Years of Service Bonus
                                            </label>
                                            <Button
                                                type="button"
                                                variant="outline"
                                                onClick={handleAddTier}
                                                className="h-8 text-xs px-3"
                                            >
                                                <Plus className="h-3 w-3 mr-1" />
                                                Add Tier
                                            </Button>
                                        </div>

                                        {serviceTiers.length > 0 ? (
                                            <div className="border border-slate-200 rounded-lg overflow-hidden">
                                                <table className="w-full text-sm">
                                                    <thead className="bg-slate-50 text-slate-600 font-medium border-b border-slate-200">
                                                        <tr>
                                                            <th className="px-4 py-2 text-left w-1/2">Service Requirement</th>
                                                            <th className="px-4 py-2 text-left w-1/2">Bonus Entitlement</th>
                                                            <th className="px-4 py-2 w-10"></th>
                                                        </tr>
                                                    </thead>
                                                    <tbody className="divide-y divide-slate-100 bg-white">
                                                        {serviceTiers.map((tier, index) => (
                                                            <tr key={index} className="group hover:bg-slate-50">
                                                                <td className="px-4 py-2">
                                                                    <div className="flex items-center gap-2">
                                                                        <span className="text-slate-500">After</span>
                                                                        <Input
                                                                            type="number"
                                                                            value={tier.years}
                                                                            onChange={(e) => handleTierChange(index, 'years', e.target.value)}
                                                                            className="w-20 h-9 text-center"
                                                                            min="1"
                                                                            placeholder="1"
                                                                        />
                                                                        <span className="text-slate-500">years</span>
                                                                    </div>
                                                                </td>
                                                                <td className="px-4 py-2">
                                                                    <div className="flex items-center gap-2">
                                                                        <span className="text-slate-500">+</span>
                                                                        <Input
                                                                            type="number"
                                                                            value={tier.bonus}
                                                                            onChange={(e) => handleTierChange(index, 'bonus', e.target.value)}
                                                                            className="w-20 h-9 text-center"
                                                                            min="0"
                                                                            placeholder="1"
                                                                        />
                                                                        <span className="text-slate-500">days</span>
                                                                    </div>
                                                                </td>
                                                                <td className="px-4 py-2 text-right">
                                                                    <button
                                                                        type="button"
                                                                        onClick={() => handleRemoveTier(index)}
                                                                        className="text-slate-400 hover:text-red-500 p-1.5 hover:bg-red-50 rounded-md transition-colors"
                                                                        title="Remove tier"
                                                                    >
                                                                        <Trash2 className="h-4 w-4" />
                                                                    </button>
                                                                </td>
                                                            </tr>
                                                        ))}
                                                    </tbody>
                                                </table>
                                            </div>
                                        ) : (
                                            <div className="text-sm text-slate-500 italic p-8 bg-slate-50 rounded-lg text-center border border-dashed border-slate-200">
                                                No service bonus tiers configured.
                                                <div className="mt-2">
                                                    Click "Add Tier" to reward long-serving employees.
                                                </div>
                                            </div>
                                        )}
                                    </div>

                                    <div className="space-y-1">
                                        <label className="text-sm font-medium text-slate-700">
                                            Max Carry Forward Days
                                        </label>
                                        <Input
                                            type="number"
                                            value={editForm.max_carry_forward_days ?? 0}
                                            onChange={(e) => setEditForm({
                                                ...editForm,
                                                max_carry_forward_days: parseInt(e.target.value) || 0
                                            })}
                                        />
                                    </div>

                                    <div className="space-y-1">
                                        <label className="text-sm font-medium text-slate-700">
                                            Minimum Advance Days
                                        </label>
                                        <Input
                                            type="number"
                                            value={editForm.min_advance_days ?? 0}
                                            onChange={(e) => setEditForm({
                                                ...editForm,
                                                min_advance_days: parseInt(e.target.value) || 0
                                            })}
                                        />
                                    </div>

                                    <div className="space-y-2">
                                        <label className="flex items-center gap-2">
                                            <input
                                                type="checkbox"
                                                checked={editForm.prorate_first_year ?? false}
                                                onChange={(e) => setEditForm({
                                                    ...editForm,
                                                    prorate_first_year: e.target.checked
                                                })}
                                                className="rounded border-slate-300"
                                            />
                                            <span className="text-sm text-slate-700">Prorate First Year</span>
                                        </label>

                                        <label className="flex items-center gap-2">
                                            <input
                                                type="checkbox"
                                                checked={editForm.allow_carry_forward ?? false}
                                                onChange={(e) => setEditForm({
                                                    ...editForm,
                                                    allow_carry_forward: e.target.checked
                                                })}
                                                className="rounded border-slate-300"
                                            />
                                            <span className="text-sm text-slate-700">Allow Carry Forward</span>
                                        </label>

                                        <label className="flex items-center gap-2">
                                            <input
                                                type="checkbox"
                                                checked={editForm.requires_attachment ?? false}
                                                onChange={(e) => setEditForm({
                                                    ...editForm,
                                                    requires_attachment: e.target.checked
                                                })}
                                                className="rounded border-slate-300"
                                            />
                                            <span className="text-sm text-slate-700">Requires Attachment</span>
                                        </label>

                                        <label className="flex items-center gap-2">
                                            <input
                                                type="checkbox"
                                                checked={editForm.is_active ?? true}
                                                onChange={(e) => setEditForm({
                                                    ...editForm,
                                                    is_active: e.target.checked
                                                })}
                                                className="rounded border-slate-300"
                                            />
                                            <span className="text-sm text-slate-700">Active</span>
                                        </label>
                                    </div>

                                    <div className="flex gap-2 pt-2">
                                        <Button
                                            onClick={() => handleSave(config.leave_type)}
                                            isLoading={saving === config.leave_type}
                                            className="flex-1"
                                        >
                                            <Save className="h-4 w-4 mr-2" />
                                            Save
                                        </Button>
                                        <Button
                                            variant="outline"
                                            onClick={handleCancel}
                                            className="flex-1"
                                        >
                                            Cancel
                                        </Button>
                                    </div>
                                </div>
                            ) : (
                                <div className="space-y-3">
                                    <div className="grid grid-cols-2 gap-4 text-sm">
                                        <div>
                                            <span className="text-slate-500">Base Entitlement:</span>
                                            <span className="ml-2 font-medium">{config.base_entitlement} days</span>
                                        </div>
                                        <div>
                                            <span className="text-slate-500">Carry Forward:</span>
                                            <span className="ml-2 font-medium">
                                                {config.allow_carry_forward ? `${config.max_carry_forward_days} days` : 'No'}
                                            </span>
                                        </div>
                                        <div>
                                            <span className="text-slate-500">Prorate:</span>
                                            <span className="ml-2">
                                                {config.prorate_first_year ? (
                                                    <Check className="h-4 w-4 text-green-600 inline" />
                                                ) : (
                                                    <X className="h-4 w-4 text-slate-400 inline" />
                                                )}
                                            </span>
                                        </div>
                                        <div>
                                            <span className="text-slate-500">Attachment:</span>
                                            <span className="ml-2">
                                                {config.requires_attachment ? (
                                                    <Check className="h-4 w-4 text-green-600 inline" />
                                                ) : (
                                                    <X className="h-4 w-4 text-slate-400 inline" />
                                                )}
                                            </span>
                                        </div>
                                    </div>

                                    {config.years_of_service_tiers && Object.keys(config.years_of_service_tiers).length > 0 && (
                                        <div className="text-sm">
                                            <span className="text-slate-500">Years of Service Bonus:</span>
                                            <div className="mt-1 flex flex-wrap gap-2">
                                                {Object.entries(config.years_of_service_tiers)
                                                    .sort(([a], [b]) => parseInt(a) - parseInt(b))
                                                    .map(([years, bonus]) => (
                                                        <span key={years} className="bg-blue-50 text-blue-700 px-2 py-0.5 rounded text-xs">
                                                            {years}+ yrs: +{bonus} days
                                                        </span>
                                                    ))}
                                            </div>
                                        </div>
                                    )}

                                    <Button
                                        variant="outline"
                                        onClick={() => handleEdit(config)}
                                        className="w-full mt-2"
                                    >
                                        Edit Configuration
                                    </Button>
                                </div>
                            )}
                        </CardContent>
                    </Card>
                ))}
            </div>

            {/* Workflow Editor Modal */}
            {workflowLeaveType && (
                <WorkflowEditor
                    leaveType={workflowLeaveType}
                    onClose={() => setWorkflowLeaveType(null)}
                />
            )}
        </div>
    );
};

export default LeaveTypeSettings;
