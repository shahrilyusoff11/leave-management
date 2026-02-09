export type UserRole = 'sysadmin' | 'admin' | 'hr' | 'manager' | 'hod' | 'staff';

export interface User {
    id: string;
    email: string;
    first_name: string;
    last_name: string;
    role: UserRole;
    department?: string;
    position?: string;
    manager_id?: string;
    joined_date: string;
    is_active: boolean;
    is_confirmed?: boolean;
    leave_entitlements?: LeaveBalance[];
}

export type LeaveType = 'annual' | 'sick' | 'maternity' | 'paternity' | 'emergency' | 'unpaid' | 'unrecorded' | 'hospitalization';
export type LeaveStatus = 'pending' | 'approved' | 'rejected' | 'cancelled';

export interface LeaveRequest {
    id: string;
    user_id: string;
    user?: User;
    leave_type: LeaveType;
    start_date: string;
    end_date: string;
    duration_days: number;
    reason: string;
    attachment_url?: string;
    status: LeaveStatus;
    approver_id?: string;
    rejection_reason?: string;
    created_at: string;
}

export interface LeaveBalance {
    id: string;
    user_id: string;
    leave_type: LeaveType;
    year: number;
    total_entitlement: number;
    used: number;
    carried_forward: number;
    adjusted: number;
    remaining: number; // Calculated, or implicit
}

// Workflow Types
export type WorkflowActionType = 'approve' | 'verify' | 'review' | 'categorize' | 'submit';
export type TimeoutAction = 'escalate' | 'auto_approve' | 'fallback_step' | 'convert_leave_type';
export type WorkflowStepAction = 'pending' | 'approved' | 'rejected' | 'verified' | 'not_verified' |
    'requested_docs' | 'escalated' | 'categorized_al' | 'categorized_unpaid' |
    'converted_to_el' | 'converted_to_unpaid' | 'timeout_applied';

export interface WorkflowStep {
    id: string;
    workflow_id: string;
    step_order: number;
    step_name: string;
    step_label: string;
    responsible_role: UserRole;
    action_type: WorkflowActionType;
    timeout_days: number;
    timeout_action: TimeoutAction;
    fallback_step_id?: string;
    convert_to_type?: LeaveType;
    conditions?: Record<string, any>;
    next_step_on_approve?: string;
    next_step_on_reject?: string;
    notify_roles?: string[];
    requires_document: boolean;
    document_type?: string;
    is_terminal: boolean;
    terminal_status?: LeaveStatus;
    created_at: string;
    updated_at: string;
}

export interface LeaveWorkflow {
    id: string;
    leave_type: LeaveType;
    workflow_name: string;
    description: string;
    first_step_id?: string;
    steps?: WorkflowStep[];
    is_active: boolean;
    created_at: string;
    updated_at: string;
}

export interface LeaveRequestWorkflowState {
    id: string;
    leave_request_id: string;
    workflow_id: string;
    current_step_id?: string;
    current_step?: WorkflowStep;
    step_started_at: string;
    previous_step_id?: string;
    action_taken: WorkflowStepAction;
    action_by?: string;
    action_comment?: string;
    step_history?: Record<string, any>;
    is_complete: boolean;
    completed_at?: string;
    final_status?: LeaveStatus;
    created_at: string;
    updated_at: string;
}

