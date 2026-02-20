import React, { useState, useEffect, useCallback } from 'react';
import ReactDOM from 'react-dom';
import api from '../services/api';
import type { LeaveWorkflow, WorkflowStep, LeaveType, UserRole, WorkflowActionType, TimeoutAction, LeaveStatus } from '../types';
import './WorkflowEditor.css';

interface WorkflowEditorProps {
    leaveType: LeaveType;
    onClose: () => void;
}

const roleOptions: UserRole[] = ['staff', 'manager', 'hod', 'hr', 'admin', 'sysadmin'];
const actionTypeOptions: WorkflowActionType[] = ['approve', 'verify', 'review', 'categorize', 'submit'];
const timeoutActionOptions: TimeoutAction[] = ['escalate', 'auto_approve', 'fallback_step', 'convert_leave_type'];

const WorkflowEditor: React.FC<WorkflowEditorProps> = ({ leaveType, onClose }) => {
    const [workflow, setWorkflow] = useState<LeaveWorkflow | null>(null);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);
    const [editingStep, setEditingStep] = useState<WorkflowStep | null>(null);
    const [isNewStep, setIsNewStep] = useState(false);
    const [saving, setSaving] = useState(false);

    const fetchWorkflow = useCallback(async () => {
        try {
            const response = await api.get<LeaveWorkflow>(`/admin/workflows/${leaveType}`);
            setWorkflow(response.data);
            setError(null);
        } catch (err: any) {
            setError(err.response?.data?.error || 'Failed to load workflow');
        } finally {
            setLoading(false);
        }
    }, [leaveType]);

    useEffect(() => {
        fetchWorkflow();
    }, [fetchWorkflow]);

    const handleStepUpdate = async (step: WorkflowStep) => {
        setSaving(true);
        try {
            await api.put(`/admin/workflows/${leaveType}/steps/${step.id}`, step);
            await fetchWorkflow();
            setEditingStep(null);
            setIsNewStep(false);
        } catch (err: any) {
            setError(err.response?.data?.error || 'Failed to update step');
        } finally {
            setSaving(false);
        }
    };

    const handleAddStep = async (step: WorkflowStep) => {
        setSaving(true);
        try {
            await api.post(`/admin/workflows/${leaveType}/steps`, {
                step_name: step.step_name,
                step_label: step.step_label,
                step_order: (workflow?.steps?.length || 0) + 1,
                responsible_role: step.responsible_role,
                action_type: step.action_type,
                timeout_days: step.timeout_days,
                timeout_action: step.timeout_action,
                requires_document: step.requires_document,
                document_type: step.document_type,
                is_terminal: step.is_terminal,
                terminal_status: step.terminal_status,
            });
            await fetchWorkflow();
            setEditingStep(null);
            setIsNewStep(false);
        } catch (err: any) {
            setError(err.response?.data?.error || 'Failed to add step');
        } finally {
            setSaving(false);
        }
    };

    const handleDeleteStep = async (stepId: string) => {
        if (!window.confirm('Are you sure you want to delete this step?')) return;

        try {
            await api.delete(`/admin/workflows/${leaveType}/steps/${stepId}`);
            await fetchWorkflow();
        } catch (err: any) {
            setError(err.response?.data?.error || 'Failed to delete step');
        }
    };

    const handleWorkflowToggle = async () => {
        if (!workflow) return;

        try {
            await api.put(`/admin/workflows/${leaveType}`, {
                is_active: !workflow.is_active
            });
            await fetchWorkflow();
        } catch (err: any) {
            setError(err.response?.data?.error || 'Failed to update workflow');
        }
    };

    const createNewStep = () => {
        const newStep: WorkflowStep = {
            id: 'new',
            workflow_id: workflow?.id || '',
            step_order: (workflow?.steps?.length || 0) + 1,
            step_name: `step_${(workflow?.steps?.length || 0) + 1}`,
            step_label: 'New Step',
            responsible_role: 'hod',
            action_type: 'approve',
            timeout_days: 7,
            timeout_action: 'fallback_step',
            requires_document: false,
            is_terminal: false,
            created_at: new Date().toISOString(),
            updated_at: new Date().toISOString(),
        };
        setEditingStep(newStep);
        setIsNewStep(true);
    };

    if (loading) {
        return ReactDOM.createPortal(
            <div className="workflow-editor-overlay">
                <div className="workflow-editor-modal">
                    <div className="loading-spinner">Loading workflow...</div>
                </div>
            </div>,
            document.body
        );
    }

    if (!workflow) {
        return ReactDOM.createPortal(
            <div className="workflow-editor-overlay">
                <div className="workflow-editor-modal">
                    <div className="workflow-header">
                        <h2>No Workflow Configured</h2>
                        <button onClick={onClose} className="close-btn">&times;</button>
                    </div>
                    <p>No workflow has been configured for this leave type.</p>
                </div>
            </div>,
            document.body
        );
    }

    return ReactDOM.createPortal(
        <>
            <div className="workflow-editor-overlay">
                <div className="workflow-editor-modal">
                    <div className="workflow-header">
                        <div>
                            <h2>{workflow.workflow_name} <span className="version-badge">v{workflow.version}</span></h2>
                            <p className="workflow-description">{workflow.description}</p>
                        </div>
                        <button onClick={onClose} className="close-btn">&times;</button>
                    </div>

                    {error && <div className="error-banner">{error}</div>}

                    <div className="workflow-status">
                        <span className={`status-badge ${workflow.is_active ? 'active' : 'inactive'}`}>
                            {workflow.is_active ? 'Active' : 'Inactive'}
                        </span>
                        <button
                            onClick={handleWorkflowToggle}
                            className={`toggle-btn ${workflow.is_active ? 'deactivate' : 'activate'}`}
                        >
                            {workflow.is_active ? 'Deactivate' : 'Activate'}
                        </button>
                    </div>

                    <div className="workflow-steps">
                        <div className="steps-header">
                            <h3>Workflow Steps</h3>
                            <button onClick={createNewStep} className="add-step-btn">
                                + Add Step
                            </button>
                        </div>
                        <div className="steps-list">
                            {workflow.steps?.sort((a, b) => a.step_order - b.step_order).map((step, index) => (
                                <div key={step.id} className="step-card">
                                    <div className="step-header">
                                        <span className="step-order">{index + 1}</span>
                                        <h4>{step.step_label || step.step_name}</h4>
                                        <div className="step-actions">
                                            <button onClick={() => { setEditingStep(step); setIsNewStep(false); }} className="edit-btn">Edit</button>
                                            <button onClick={() => handleDeleteStep(step.id)} className="delete-btn">Delete</button>
                                        </div>
                                    </div>
                                    <div className="step-details">
                                        <div className="detail-row">
                                            <span className="label">Role:</span>
                                            <span className="value role-badge">{step.responsible_role}</span>
                                        </div>
                                        <div className="detail-row">
                                            <span className="label">Action:</span>
                                            <span className="value">{step.action_type}</span>
                                        </div>
                                        <div className="detail-row">
                                            <span className="label">Timeout:</span>
                                            <span className="value">{step.timeout_days} days → {step.timeout_action}</span>
                                        </div>
                                        {step.requires_document && (
                                            <div className="detail-row">
                                                <span className="label">Document:</span>
                                                <span className="value">{step.document_type || 'Required'}</span>
                                            </div>
                                        )}
                                        {step.is_terminal && (
                                            <div className="detail-row terminal">
                                                <span className="label">Terminal:</span>
                                                <span className="value">{step.terminal_status}</span>
                                            </div>
                                        )}
                                    </div>
                                </div>
                            ))}
                            {(!workflow.steps || workflow.steps.length === 0) && (
                                <div className="no-steps">
                                    <p>No steps configured. Click "Add Step" to create a workflow step.</p>
                                </div>
                            )}
                        </div>
                    </div>
                </div>
            </div>

            {editingStep && (
                <StepEditor
                    step={editingStep}
                    allSteps={workflow.steps || []}
                    onSave={isNewStep ? handleAddStep : handleStepUpdate}
                    onCancel={() => { setEditingStep(null); setIsNewStep(false); }}
                    saving={saving}
                    isNew={isNewStep}
                />
            )}
        </>,
        document.body
    );
};

interface StepEditorProps {
    step: WorkflowStep;
    allSteps: WorkflowStep[];
    onSave: (step: WorkflowStep) => void;
    onCancel: () => void;
    saving: boolean;
    isNew?: boolean;
}

const StepEditor: React.FC<StepEditorProps> = ({ step, allSteps, onSave, onCancel, saving, isNew }) => {
    const [editedStep, setEditedStep] = useState<WorkflowStep>(step);

    const handleChange = (field: keyof WorkflowStep, value: any) => {
        setEditedStep(prev => ({ ...prev, [field]: value }));
    };

    // StepEditor is already portalled because the parent is portalling the fragment containing it
    return (
        <div className="step-editor-overlay">
            <div className="step-editor-modal">
                <h3>{isNew ? 'Add New Step' : `Edit Step: ${step.step_label || step.step_name}`}</h3>

                <div className="form-group">
                    <label>Step Name (internal)</label>
                    <input
                        type="text"
                        value={editedStep.step_name}
                        onChange={e => handleChange('step_name', e.target.value)}
                        placeholder="e.g., hod_approval"
                    />
                </div>

                <div className="form-group">
                    <label>Step Label (display)</label>
                    <input
                        type="text"
                        value={editedStep.step_label}
                        onChange={e => handleChange('step_label', e.target.value)}
                        placeholder="e.g., HOD Approval"
                    />
                </div>
                <div className="form-group">
                    <label>Responsible Role</label>
                    <select
                        value={editedStep.responsible_role}
                        onChange={e => handleChange('responsible_role', e.target.value as UserRole)}
                    >
                        {roleOptions.map(role => (
                            <option key={role} value={role}>{role}</option>
                        ))}
                    </select>
                </div>

                <div className="form-group">
                    <label>Action Type</label>
                    <select
                        value={editedStep.action_type}
                        onChange={e => handleChange('action_type', e.target.value as WorkflowActionType)}
                    >
                        {actionTypeOptions.map(action => (
                            <option key={action} value={action}>{action}</option>
                        ))}
                    </select>
                </div>

                <div className="form-row">
                    <div className="form-group">
                        <label>Timeout (days)</label>
                        <input
                            type="number"
                            min={1}
                            value={editedStep.timeout_days}
                            onChange={e => handleChange('timeout_days', parseInt(e.target.value))}
                        />
                    </div>

                    <div className="form-group">
                        <label>Timeout Action</label>
                        <select
                            value={editedStep.timeout_action}
                            onChange={e => handleChange('timeout_action', e.target.value as TimeoutAction)}
                        >
                            {timeoutActionOptions.map(action => (
                                <option key={action} value={action}>{action.replace(/_/g, ' ')}</option>
                            ))}
                        </select>
                    </div>
                </div>

                <div className="form-row">
                    <div className="form-group">
                        <label>Next Step on Approve</label>
                        <select
                            value={editedStep.next_step_on_approve || ''}
                            onChange={e => handleChange('next_step_on_approve', e.target.value || null)}
                        >
                            <option value="">None (End)</option>
                            {allSteps.filter(s => s.id !== step.id).map(s => (
                                <option key={s.id} value={s.id}>{s.step_label || s.step_name}</option>
                            ))}
                        </select>
                    </div>

                    <div className="form-group">
                        <label>Next Step on Reject</label>
                        <select
                            value={editedStep.next_step_on_reject || ''}
                            onChange={e => handleChange('next_step_on_reject', e.target.value || null)}
                        >
                            <option value="">None (End)</option>
                            {allSteps.filter(s => s.id !== step.id).map(s => (
                                <option key={s.id} value={s.id}>{s.step_label || s.step_name}</option>
                            ))}
                        </select>
                    </div>
                </div>

                <div className="form-group checkbox-group">
                    <label>
                        <input
                            type="checkbox"
                            checked={editedStep.requires_document}
                            onChange={e => handleChange('requires_document', e.target.checked)}
                        />
                        Requires Document
                    </label>
                </div>

                {editedStep.requires_document && (
                    <div className="form-group">
                        <label>Document Type</label>
                        <input
                            type="text"
                            value={editedStep.document_type || ''}
                            onChange={e => handleChange('document_type', e.target.value)}
                            placeholder="e.g., medical_certificate"
                        />
                    </div>
                )}

                <div className="form-group checkbox-group">
                    <label>
                        <input
                            type="checkbox"
                            checked={editedStep.is_terminal}
                            onChange={e => handleChange('is_terminal', e.target.checked)}
                        />
                        Terminal Step (ends workflow)
                    </label>
                </div>

                {editedStep.is_terminal && (
                    <div className="form-group">
                        <label>Terminal Status</label>
                        <select
                            value={editedStep.terminal_status || 'approved'}
                            onChange={e => handleChange('terminal_status', e.target.value as LeaveStatus)}
                        >
                            <option value="approved">Approved</option>
                            <option value="rejected">Rejected</option>
                            <option value="cancelled">Cancelled</option>
                        </select>
                    </div>
                )}

                <div className="form-actions">
                    <button onClick={onCancel} disabled={saving} className="cancel-btn">Cancel</button>
                    <button onClick={() => onSave(editedStep)} disabled={saving} className="save-btn">
                        {saving ? 'Saving...' : isNew ? 'Add Step' : 'Save Changes'}
                    </button>
                </div>
            </div>
        </div>
    );
};

export default WorkflowEditor;

