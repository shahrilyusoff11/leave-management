import React, { useEffect, useState } from 'react';
import { useForm } from 'react-hook-form';
import { Plus, Search, MoreVertical, Shield, UserCheck, UserX } from 'lucide-react';
import api from '../services/api';
import type { User, UserRole, Department } from '../types';
import { Card } from '../components/ui/Card';
import { Button } from '../components/ui/Button';
import { Input } from '../components/ui/Input';
import { Modal } from '../components/ui/Modal';
import { format } from 'date-fns';
import { useToast } from '../components/ui/Toast';
import { useAuth } from '../context/AuthContext';
import { getRoleLabel, getRoleBadgeColor, getAssignableRoles } from '../utils/roles';

const UserManagement: React.FC = () => {
    const { user: currentUser } = useAuth();
    const [users, setUsers] = useState<User[]>([]);
    const [loading, setLoading] = useState(true);
    const [isCreateModalOpen, setIsCreateModalOpen] = useState(false);
    const [editUser, setEditUser] = useState<User | null>(null);
    const [searchQuery, setSearchQuery] = useState('');
    const [roleFilter, setRoleFilter] = useState<string>('all');

    const fetchUsers = async () => {
        setLoading(true);
        try {
            const response = await api.get('/hr/users');
            if (Array.isArray(response.data)) {
                setUsers(response.data);
            }
        } catch (error) {
            console.error("Failed to fetch users", error);
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => {
        fetchUsers();
    }, []);

    const filteredUsers = users.filter(user => {
        const matchesSearch = user.email.toLowerCase().includes(searchQuery.toLowerCase()) ||
            user.first_name.toLowerCase().includes(searchQuery.toLowerCase()) ||
            user.last_name.toLowerCase().includes(searchQuery.toLowerCase());
        const matchesRole = roleFilter === 'all' || user.role === roleFilter;
        return matchesSearch && matchesRole;
    });

    return (
        <div className="space-y-6">
            <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4">
                <div>
                    <h1 className="text-2xl font-bold text-slate-900">User Management</h1>
                    <p className="text-slate-500 mt-1">Manage system users and access controls</p>
                </div>
                <Button onClick={() => setIsCreateModalOpen(true)}>
                    <Plus className="h-4 w-4 mr-2" />
                    Add User
                </Button>
            </div>

            <Card className="p-4">
                <div className="flex items-center gap-3 mb-4">
                    <div className="relative flex-1 max-w-sm">
                        <Search className="absolute left-3 top-2.5 h-4 w-4 text-slate-400" />
                        <Input
                            placeholder="Search users..."
                            className="pl-9"
                            value={searchQuery}
                            onChange={(e) => setSearchQuery(e.target.value)}
                        />
                    </div>
                    <select
                        value={roleFilter}
                        onChange={(e) => setRoleFilter(e.target.value)}
                        className="h-10 px-3 rounded-lg border border-slate-200 bg-white text-sm focus:outline-none focus:ring-2 focus:ring-brand-500/20 focus:border-brand-500"
                    >
                        <option value="all">All Roles</option>
                        <option value="sysadmin">System Admin</option>
                        <option value="admin">Administrator</option>
                        <option value="hr">Human Resources</option>
                        <option value="hod">Head of Department</option>
                        <option value="manager">Manager</option>
                        <option value="staff">Staff</option>
                    </select>
                    <div className="text-sm text-slate-500">
                        {filteredUsers.length} user{filteredUsers.length !== 1 ? 's' : ''}
                    </div>
                </div>

                <div className="overflow-x-auto">
                    <table className="w-full text-left text-sm">
                        <thead>
                            <tr className="bg-slate-50 border-b border-slate-200">
                                <th className="px-6 py-4 font-semibold text-slate-600">User</th>
                                <th className="px-6 py-4 font-semibold text-slate-600">Role</th>
                                <th className="px-6 py-4 font-semibold text-slate-600">Department</th>
                                <th className="px-6 py-4 font-semibold text-slate-600">Status</th>
                                <th className="px-6 py-4 font-semibold text-slate-600">Joined</th>
                                <th className="px-6 py-4 font-semibold text-slate-600 text-right">Actions</th>
                            </tr>
                        </thead>
                        <tbody className="divide-y divide-slate-100">
                            {loading ? (
                                <tr>
                                    <td colSpan={6} className="px-6 py-12 text-center text-slate-500">
                                        Loading users...
                                    </td>
                                </tr>
                            ) : filteredUsers.length === 0 ? (
                                <tr>
                                    <td colSpan={6} className="px-6 py-12 text-center text-slate-500">
                                        No users found
                                    </td>
                                </tr>
                            ) : (
                                filteredUsers.map((user) => (
                                    <tr key={user.id} className="hover:bg-slate-50 transition-colors">
                                        <td className="px-6 py-4">
                                            <div className="flex items-center gap-3">
                                                <div className="h-9 w-9 rounded-full bg-slate-100 flex items-center justify-center text-slate-600 font-bold">
                                                    {user.first_name[0]}{user.last_name[0]}
                                                </div>
                                                <div>
                                                    <p className="font-medium text-slate-900">{user.first_name} {user.last_name}</p>
                                                    <p className="text-xs text-slate-500">{user.email}</p>
                                                </div>
                                            </div>
                                        </td>
                                        <td className="px-6 py-4">
                                            <span className={`inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-medium ${getRoleBadgeColor(user.role as UserRole)}`}>
                                                <Shield className="h-3 w-3" />
                                                {getRoleLabel(user.role as UserRole)}
                                            </span>
                                        </td>
                                        <td className="px-6 py-4 text-slate-600">
                                            {user.department_ref?.name || '-'}
                                        </td>
                                        <td className="px-6 py-4">
                                            {user.is_active !== false ? (
                                                <span className="inline-flex items-center gap-1 text-xs font-medium text-green-700 bg-green-50 px-2 py-0.5 rounded-full">
                                                    <UserCheck className="h-3 w-3" /> Active
                                                </span>
                                            ) : (
                                                <span className="inline-flex items-center gap-1 text-xs font-medium text-red-700 bg-red-50 px-2 py-0.5 rounded-full">
                                                    <UserX className="h-3 w-3" /> Inactive
                                                </span>
                                            )}
                                        </td>
                                        <td className="px-6 py-4 text-slate-600">
                                            {format(new Date(user.joined_date), 'MMM d, yyyy')}
                                        </td>
                                        <td className="px-6 py-4 text-right">
                                            <Button
                                                variant="ghost"
                                                size="sm"
                                                className="h-8 w-8 p-0"
                                                onClick={() => setEditUser(user)}
                                            >
                                                <MoreVertical className="h-4 w-4 text-slate-400" />
                                            </Button>
                                        </td>
                                    </tr>
                                ))
                            )}
                        </tbody>
                    </table>
                </div>
            </Card>

            <CreateUserModal
                isOpen={isCreateModalOpen}
                onClose={() => setIsCreateModalOpen(false)}
                onSuccess={fetchUsers}
                currentUserRole={currentUser?.role as UserRole}
            />

            {editUser && (
                <EditUserModal
                    user={editUser}
                    isOpen={!!editUser}
                    onClose={() => setEditUser(null)}
                    onSuccess={fetchUsers}
                    currentUserRole={currentUser?.role as UserRole}
                />
            )}
        </div>
    );
};

const CreateUserModal = ({ isOpen, onClose, onSuccess, currentUserRole }: { isOpen: boolean, onClose: () => void, onSuccess: () => void, currentUserRole?: UserRole }) => {
    const { register, handleSubmit, formState: { errors, isSubmitting }, reset } = useForm();
    const [error, setError] = useState('');
    const [managers, setManagers] = useState<User[]>([]);
    const [departments, setDepartments] = useState<Department[]>([]);

    useEffect(() => {
        const fetchData = async () => {
            try {
                const [usersRes, deptsRes] = await Promise.all([
                    api.get('/hr/users'),
                    api.get('/hr/departments')
                ]);
                if (Array.isArray(usersRes.data)) {
                    const potentialManagers = usersRes.data.filter((u: User) =>
                        ['manager', 'hod', 'hr', 'admin', 'sysadmin'].includes(u.role)
                    );
                    setManagers(potentialManagers);
                }
                if (Array.isArray(deptsRes.data)) {
                    setDepartments(deptsRes.data);
                }
            } catch (err) {
                console.error("Failed to fetch data", err);
            }
        };
        if (isOpen) {
            fetchData();
        }
    }, [isOpen]);

    const onSubmit = async (data: any) => {
        setError('');
        try {
            const payload = {
                ...data,
                manager_id: data.manager_id || null
            };
            await api.post('/hr/users', payload);
            reset();
            onSuccess();
            onClose();
        } catch (err: any) {
            setError(err.response?.data?.error || "Failed to create user");
        }
    };

    return (
        <Modal isOpen={isOpen} onClose={onClose} title="Add New User">
            <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
                {error && (
                    <div className="p-3 text-sm bg-red-50 text-red-600 rounded-lg">
                        {error}
                    </div>
                )}

                <div className="grid grid-cols-2 gap-4">
                    <div className="space-y-1">
                        <label className="text-sm font-medium text-slate-700">First Name</label>
                        <Input {...register('first_name', { required: 'Required' })} error={errors.first_name?.message as string} />
                    </div>
                    <div className="space-y-1">
                        <label className="text-sm font-medium text-slate-700">Last Name</label>
                        <Input {...register('last_name', { required: 'Required' })} error={errors.last_name?.message as string} />
                    </div>
                </div>

                <div className="space-y-1">
                    <label className="text-sm font-medium text-slate-700">Email</label>
                    <Input type="email" {...register('email', { required: 'Required' })} error={errors.email?.message as string} />
                </div>

                <div className="space-y-1">
                    <label className="text-sm font-medium text-slate-700">Password</label>
                    <Input type="password" {...register('password', { required: 'Required', minLength: { value: 8, message: 'Min 8 chars' } })} error={errors.password?.message as string} />
                </div>

                <div className="grid grid-cols-2 gap-4">
                    <div className="space-y-1">
                        <label className="text-sm font-medium text-slate-700">Role</label>
                        <select
                            className="w-full h-10 px-3 rounded-lg border border-slate-200 bg-white text-sm focus:outline-none focus:ring-2 focus:ring-brand-500/20 focus:border-brand-500"
                            {...register('role', { required: 'Required' })}
                        >
                            {getAssignableRoles(currentUserRole).map(role => (
                                <option key={role} value={role}>{getRoleLabel(role)}</option>
                            ))}
                        </select>
                    </div>
                    <div className="space-y-1">
                        <label className="text-sm font-medium text-slate-700">Joined Date</label>
                        <Input type="date" {...register('joined_date', { required: 'Required' })} error={errors.joined_date?.message as string} />
                    </div>
                </div>

                <div className="grid grid-cols-2 gap-4">
                    <div className="space-y-1">
                        <label className="text-sm font-medium text-slate-700">Department</label>
                        <select
                            className="w-full h-10 px-3 rounded-lg border border-slate-200 bg-white text-sm focus:outline-none focus:ring-2 focus:ring-brand-500/20 focus:border-brand-500"
                            {...register('department_id')}
                        >
                            <option value="">No Department</option>
                            {departments.map(d => (
                                <option key={d.id} value={d.id}>{d.name}</option>
                            ))}
                        </select>
                    </div>
                    <div className="space-y-1">
                        <label className="text-sm font-medium text-slate-700">Position</label>
                        <Input {...register('position', { required: 'Required' })} error={errors.position?.message as string} />
                    </div>
                </div>

                <div className="space-y-1">
                    <label className="text-sm font-medium text-slate-700">Reporting Manager</label>
                    <select
                        className="w-full h-10 px-3 rounded-lg border border-slate-200 bg-white text-sm focus:outline-none focus:ring-2 focus:ring-brand-500/20 focus:border-brand-500"
                        {...register('manager_id')}
                    >
                        <option value="">No Manager (Routes to System Admin)</option>
                        {managers.map(m => (
                            <option key={m.id} value={m.id}>
                                {m.first_name} {m.last_name} — {getRoleLabel(m.role as UserRole)}{m.department_ref ? ` (${m.department_ref.name})` : ''}
                            </option>
                        ))}
                    </select>
                    <p className="text-xs text-slate-400">Leave requests will be routed through the workflow for this manager's department</p>
                </div>

                <div className="flex justify-end gap-3 mt-6">
                    <Button type="button" variant="ghost" onClick={onClose}>Cancel</Button>
                    <Button type="submit" isLoading={isSubmitting}>Create User</Button>
                </div>
            </form>
        </Modal>
    );
};

const EditUserModal = ({ user, isOpen, onClose, onSuccess, currentUserRole }: { user: User, isOpen: boolean, onClose: () => void, onSuccess: () => void, currentUserRole?: UserRole }) => {
    const { showToast } = useToast();
    const [activeTab, setActiveTab] = useState<'details' | 'probation' | 'balance' | 'history'>('details');
    const [message, setMessage] = useState('');
    const [error, setError] = useState('');
    const [isActive, setIsActive] = useState(user.is_active !== false);
    const [managers, setManagers] = useState<User[]>([]);

    const [fullUser, setFullUser] = useState<User>(user);
    const [loadingDetails, setLoadingDetails] = useState(false);
    const [departments, setDepartments] = useState<Department[]>([]);

    const [leaveHistory, setLeaveHistory] = useState<any[]>([]);
    const [loadingHistory, setLoadingHistory] = useState(false);

    const fetchUserAndManagers = async () => {
        setLoadingDetails(true);
        try {
            const [userFullResponse, allUsersResponse, deptsResponse] = await Promise.all([
                api.get(`/hr/users/${user.id}`),
                api.get('/hr/users'),
                api.get('/hr/departments')
            ]);

            const userData = userFullResponse.data;
            setFullUser(userData);

            if (Array.isArray(allUsersResponse.data)) {
                const potentialManagers = allUsersResponse.data.filter((u: User) =>
                    ['manager', 'hod', 'hr', 'admin', 'sysadmin'].includes(u.role) && u.id !== user.id
                );
                setManagers(potentialManagers);
            }

            if (Array.isArray(deptsResponse.data)) {
                setDepartments(deptsResponse.data);
            }

            // Update form with latest details
            // Doing this after fetching managers ensures the select options are ready
            // (or at least collected in the same render batch)
            detailsForm.reset({
                first_name: userData.first_name,
                last_name: userData.last_name,
                role: userData.role,
                department_id: userData.department_id || '',
                position: userData.position || '',
                manager_id: userData.manager_id || '',
            });

        } catch (err) {
            console.error("Failed to fetch user details or managers", err);
        } finally {
            setLoadingDetails(false);
        }
    };

    useEffect(() => {
        if (isOpen) {
            fetchUserAndManagers();
        }
    }, [isOpen, user.id]);

    useEffect(() => {
        if (isOpen && activeTab === 'history') {
            const fetchHistory = async () => {
                setLoadingHistory(true);
                try {
                    const response = await api.get(`/hr/users/${user.id}/leaves`);
                    if (Array.isArray(response.data)) {
                        setLeaveHistory(response.data);
                    }
                } catch (err) {
                    console.error("Failed to fetch leave history", err);
                } finally {
                    setLoadingHistory(false);
                }
            };
            fetchHistory();
        }
    }, [isOpen, activeTab, user.id]);

    const detailsForm = useForm({
        defaultValues: {
            first_name: user.first_name,
            last_name: user.last_name,
            role: user.role,
            department_id: user.department_id || '',
            position: user.position || '',
            manager_id: (user as any).manager_id || '',
        }
    });
    const probationForm = useForm();
    const balanceForm = useForm();

    const handleUpdateDetails = async (data: any) => {
        setError('');
        setMessage('');
        try {
            await api.put(`/hr/users/${user.id}`, {
                first_name: data.first_name,
                last_name: data.last_name,
                role: data.role,
                department_id: data.department_id || null,
                position: data.position,
                manager_id: data.manager_id || null,
            });
            onSuccess();
            onClose();
            showToast('User updated successfully', 'success');
        } catch (err: any) {
            setError(err.response?.data?.error || "Failed to update user");
        }
    };

    const handleToggleActive = async () => {
        setError('');
        setMessage('');
        try {
            await api.put(`/hr/users/${user.id}/status`, {
                is_active: !isActive
            });
            setIsActive(!isActive);
            setMessage(`User ${!isActive ? 'activated' : 'deactivated'} successfully`);
            onSuccess();
        } catch (err: any) {
            setError(err.response?.data?.error || "Failed to update user status");
        }
    };

    const handleConfirmProbation = async (data: any) => {
        setError('');
        setMessage('');
        try {
            await api.put(`/hr/users/${user.id}/probation`, {
                is_confirmed: true,
                notes: data.notes
            });
            setMessage('Probation confirmed successfully');
            onSuccess();
            // Refresh details to show confirmed state
            fetchUserAndManagers();
        } catch (err: any) {
            setError(err.response?.data?.error || "Failed to update probation");
        }
    };

    const handleUpdateBalance = async (data: any) => {
        setError('');
        setMessage('');
        try {
            await api.put(`/hr/users/${user.id}/leave-balance`, {
                leave_type: data.leave_type,
                year: parseInt(data.year),
                total_entitlement: parseFloat(data.total_entitlement),
                adjustment: parseFloat(data.adjustment || 0),
                reason: data.reason
            });
            setMessage('Leave balance updated successfully');
            balanceForm.reset();
            onSuccess(); // Updates the parent list
            fetchUserAndManagers(); // Reloads the current user details (table)
        } catch (err: any) {
            setError(err.response?.data?.error || "Failed to update balance");
        }
    };

    return (
        <Modal isOpen={isOpen} onClose={onClose} title={`Edit User: ${user.first_name} ${user.last_name}`} className="max-w-4xl" position="top">
            <div className="space-y-4">
                <div className="flex border-b border-slate-200">
                    <button
                        className={`px-4 py-2 text-sm font-medium border-b-2 transition-colors ${activeTab === 'details' ? 'border-brand-500 text-brand-600' : 'border-transparent text-slate-500 hover:text-slate-700'}`}
                        onClick={() => setActiveTab('details')}
                    >
                        Details
                    </button>
                    <button
                        className={`px-4 py-2 text-sm font-medium border-b-2 transition-colors ${activeTab === 'probation' ? 'border-brand-500 text-brand-600' : 'border-transparent text-slate-500 hover:text-slate-700'}`}
                        onClick={() => setActiveTab('probation')}
                    >
                        Probation
                    </button>
                    <button
                        className={`px-4 py-2 text-sm font-medium border-b-2 transition-colors ${activeTab === 'balance' ? 'border-brand-500 text-brand-600' : 'border-transparent text-slate-500 hover:text-slate-700'}`}
                        onClick={() => setActiveTab('balance')}
                    >
                        Leave Balance
                    </button>
                    <button
                        className={`px-4 py-2 text-sm font-medium border-b-2 transition-colors ${activeTab === 'history' ? 'border-brand-500 text-brand-600' : 'border-transparent text-slate-500 hover:text-slate-700'}`}
                        onClick={() => setActiveTab('history')}
                    >
                        Leave History
                    </button>
                </div>

                {message && <div className="p-3 bg-green-50 text-green-700 rounded-lg text-sm">{message}</div>}
                {error && <div className="p-3 bg-red-50 text-red-700 rounded-lg text-sm">{error}</div>}

                {activeTab === 'details' && (
                    <form onSubmit={detailsForm.handleSubmit(handleUpdateDetails)} className="space-y-4">
                        <div className="grid grid-cols-2 gap-4">
                            <div className="space-y-1">
                                <label className="text-sm font-medium text-slate-700">First Name</label>
                                <input
                                    {...detailsForm.register("first_name", { required: true })}
                                    className="w-full px-3 py-2 border border-slate-300 rounded-lg focus:ring-2 focus:ring-brand-500 focus:border-brand-500"
                                />
                            </div>
                            <div className="space-y-1">
                                <label className="text-sm font-medium text-slate-700">Last Name</label>
                                <input
                                    {...detailsForm.register("last_name", { required: true })}
                                    className="w-full px-3 py-2 border border-slate-300 rounded-lg focus:ring-2 focus:ring-brand-500 focus:border-brand-500"
                                />
                            </div>
                        </div>
                        <div className="space-y-1">
                            <label className="text-sm font-medium text-slate-700">Email</label>
                            <input
                                value={user.email}
                                disabled
                                className="w-full px-3 py-2 border border-slate-200 rounded-lg bg-slate-50 text-slate-500"
                            />
                            <p className="text-xs text-slate-400">Email cannot be changed</p>
                        </div>
                        <div className="grid grid-cols-2 gap-4">
                            <div className="space-y-1">
                                <label className="text-sm font-medium text-slate-700">Role</label>
                                <select
                                    {...detailsForm.register("role", { required: true })}
                                    className="w-full px-3 py-2 border border-slate-300 rounded-lg focus:ring-2 focus:ring-brand-500 focus:border-brand-500"
                                >
                                    {getAssignableRoles(currentUserRole).map(role => (
                                        <option key={role} value={role}>{getRoleLabel(role)}</option>
                                    ))}
                                </select>
                            </div>
                            <div className="space-y-1">
                                <label className="text-sm font-medium text-slate-700">Department</label>
                                <select
                                    {...detailsForm.register("department_id")}
                                    className="w-full px-3 py-2 border border-slate-300 rounded-lg focus:ring-2 focus:ring-brand-500 focus:border-brand-500"
                                >
                                    <option value="">No Department</option>
                                    {departments.map(d => (
                                        <option key={d.id} value={d.id}>{d.name}</option>
                                    ))}
                                </select>
                            </div>
                        </div>
                        <div className="space-y-1">
                            <label className="text-sm font-medium text-slate-700">Position</label>
                            <input
                                {...detailsForm.register("position")}
                                className="w-full px-3 py-2 border border-slate-300 rounded-lg focus:ring-2 focus:ring-brand-500 focus:border-brand-500"
                            />
                        </div>
                        <div className="space-y-1">
                            <label className="text-sm font-medium text-slate-700">Reporting Manager</label>
                            <select
                                {...detailsForm.register("manager_id")}
                                className="w-full px-3 py-2 border border-slate-300 rounded-lg focus:ring-2 focus:ring-brand-500 focus:border-brand-500"
                            >
                                <option value="">No Manager (Routes to System Admin)</option>
                                {managers.map(m => (
                                    <option key={m.id} value={m.id}>
                                        {m.first_name} {m.last_name} — {getRoleLabel(m.role as UserRole)}{m.department_ref ? ` (${m.department_ref.name})` : ''}
                                    </option>
                                ))}
                            </select>
                            <p className="text-xs text-slate-400">Leave requests will be routed through the workflow for this manager's department</p>
                        </div>
                        <div className="space-y-1">
                            <label className="text-sm font-medium text-slate-700">Joined Date</label>
                            <div className="px-3 py-2 border border-slate-200 rounded-lg bg-slate-50 text-slate-600">
                                {format(new Date(user.joined_date), 'MMM d, yyyy')}
                            </div>
                        </div>
                        <div className="flex items-center justify-between p-3 bg-slate-50 rounded-lg">
                            <div>
                                <div className="text-sm font-medium text-slate-700">Account Status</div>
                                <div className="text-xs text-slate-500">
                                    {isActive ? 'User can log in and use the system' : 'User is deactivated and cannot log in'}
                                </div>
                            </div>
                            <button
                                type="button"
                                onClick={handleToggleActive}
                                className={`px-4 py-2 rounded-lg text-sm font-medium transition-colors ${isActive
                                    ? 'bg-red-100 text-red-700 hover:bg-red-200'
                                    : 'bg-green-100 text-green-700 hover:bg-green-200'}`}
                            >
                                {isActive ? 'Deactivate' : 'Activate'}
                            </button>
                        </div>
                        <button
                            type="submit"
                            className="w-full py-2.5 bg-brand-600 text-white rounded-lg hover:bg-brand-700 transition-colors font-medium"
                        >
                            Save Changes
                        </button>
                    </form>
                )}

                {activeTab === 'probation' && (
                    fullUser.is_confirmed ? (
                        <div className="p-4 bg-green-50 rounded-lg border border-green-200 text-center">
                            <p className="text-green-800 font-medium">Probation Confirmed</p>
                            <p className="text-green-600 text-sm mt-1">This user has successfully completed their probation period.</p>
                        </div>
                    ) : (
                        <form onSubmit={probationForm.handleSubmit(handleConfirmProbation)} className="space-y-4">
                            <p className="text-sm text-slate-600">Confirm successful completion of probation period.</p>
                            <div className="space-y-1">
                                <label className="text-sm font-medium text-slate-700">Notes (Optional)</label>
                                <Input {...probationForm.register('notes')} placeholder="Performance review notes..." />
                            </div>
                            <Button type="submit" isLoading={probationForm.formState.isSubmitting}>Confirm Probation</Button>
                        </form>
                    )
                )}

                {activeTab === 'balance' && (
                    <div className="space-y-6">
                        {loadingDetails ? (
                            <div className="text-center py-4 text-slate-500">Loading leave balances...</div>
                        ) : (
                            <div className="border border-slate-200 rounded-lg overflow-hidden">
                                <table className="w-full text-sm text-left">
                                    <thead className="bg-slate-50 text-slate-600 font-medium">
                                        <tr>
                                            <th className="px-4 py-3">Leave Type</th>
                                            <th className="px-4 py-3">Year</th>
                                            <th className="px-4 py-3 text-right">Base Entitlement</th>
                                            <th className="px-4 py-3 text-right">Adjustment</th>
                                            <th className="px-4 py-3 text-right">Total Entitlement</th>
                                            <th className="px-4 py-3 text-right">Used</th>
                                            <th className="px-4 py-3 text-right">Remaining</th>
                                        </tr>
                                    </thead>
                                    <tbody className="divide-y divide-slate-100">
                                        {fullUser.leave_entitlements?.map((balance) => {
                                            // Calculate actual totals
                                            const total = (balance.total_entitlement || 0) + (balance.adjusted || 0) + (balance.carried_forward || 0);
                                            const remaining = total - (balance.used || 0);

                                            return (
                                                <tr
                                                    key={balance.id}
                                                    className="hover:bg-slate-50 cursor-pointer transition-colors"
                                                    onClick={() => {
                                                        balanceForm.setValue('leave_type', balance.leave_type);
                                                        balanceForm.setValue('year', balance.year);
                                                        balanceForm.setValue('total_entitlement', balance.total_entitlement);
                                                        balanceForm.setValue('adjustment', balance.adjusted);
                                                    }}
                                                >
                                                    <td className="px-4 py-3 capitalize">{balance.leave_type}</td>
                                                    <td className="px-4 py-3">{balance.year}</td>
                                                    <td className="px-4 py-3 text-right">{balance.total_entitlement}</td>
                                                    <td className="px-4 py-3 text-right">
                                                        {balance.adjusted > 0 ? `+${balance.adjusted}` : balance.adjusted}
                                                    </td>
                                                    <td className="px-4 py-3 text-right font-medium">{total}</td>
                                                    <td className="px-4 py-3 text-right">{balance.used}</td>
                                                    <td className="px-4 py-3 text-right font-medium text-brand-600">{remaining}</td>
                                                </tr>
                                            )
                                        })}
                                        {(!fullUser.leave_entitlements || fullUser.leave_entitlements.length === 0) && (
                                            <tr>
                                                <td colSpan={7} className="px-4 py-6 text-center text-slate-500">
                                                    No leave balances found for this user.
                                                </td>
                                            </tr>
                                        )}
                                    </tbody>
                                </table>
                            </div>
                        )}

                        <form onSubmit={balanceForm.handleSubmit(handleUpdateBalance)} className="space-y-4 pt-4 border-t border-slate-200">
                            <h3 className="font-medium text-slate-900">Update Balance</h3>
                            <div className="grid grid-cols-2 gap-4">
                                <div className="space-y-1">
                                    <label className="text-sm font-medium text-slate-700">Leave Type</label>
                                    <select
                                        className="w-full h-10 px-3 rounded-lg border border-slate-300 bg-white text-sm focus:outline-none focus:ring-2 focus:ring-brand-500/20 focus:border-brand-500"
                                        {...balanceForm.register('leave_type', { required: true })}
                                    >
                                        <option value="annual">Annual</option>
                                        <option value="sick">Sick</option>
                                        <option value="maternity">Maternity</option>
                                        <option value="paternity">Paternity</option>
                                        <option value="emergency">Emergency</option>
                                        <option value="unpaid">Unpaid</option>
                                        <option value="hospitalization">Hospitalization</option>
                                        <option value="unrecorded">Unrecorded</option>
                                    </select>
                                </div>
                                <div className="space-y-1">
                                    <label className="text-sm font-medium text-slate-700">Year</label>
                                    <Input type="number" {...balanceForm.register('year', { required: true })} defaultValue={new Date().getFullYear()} />
                                </div>
                            </div>

                            <div className="grid grid-cols-2 gap-4">
                                <div className="space-y-1">
                                    <label className="text-sm font-medium text-slate-700">Total Entitlement</label>
                                    <Input type="number" step="0.5" {...balanceForm.register('total_entitlement', { required: true })} placeholder="e.g. 14" />
                                </div>
                                <div className="space-y-1">
                                    <label className="text-sm font-medium text-slate-700">Manual Adjustment</label>
                                    <Input type="number" step="0.5" {...balanceForm.register('adjustment')} placeholder="e.g. +1 or -1" />
                                </div>
                            </div>

                            <div className="space-y-1">
                                <label className="text-sm font-medium text-slate-700">Reason</label>
                                <Input {...balanceForm.register('reason', { required: true })} placeholder="Reason for change..." />
                            </div>

                            <Button type="submit" isLoading={balanceForm.formState.isSubmitting}>Update Balance</Button>
                        </form>
                    </div>
                )}

                {activeTab === 'history' && (
                    <div className="space-y-4">
                        <div className="flex justify-between items-center mb-4">
                            <h3 className="font-medium text-slate-900">Leave History</h3>
                            <button onClick={() => {
                                // Re-trigger fetch
                                setLoadingHistory(true);
                                api.get(`/hr/users/${user.id}/leaves`).then(res => {
                                    if (Array.isArray(res.data)) setLeaveHistory(res.data);
                                }).finally(() => setLoadingHistory(false));
                            }} className="text-sm text-brand-600 hover:text-brand-700 font-medium">
                                Refresh
                            </button>
                        </div>

                        {loadingHistory ? (
                            <div className="text-center py-8 text-slate-500">Loading history...</div>
                        ) : leaveHistory.length === 0 ? (
                            <div className="text-center py-8 text-slate-500 border border-slate-200 rounded-lg">No leave history found for this user.</div>
                        ) : (
                            <div className="border border-slate-200 rounded-lg overflow-hidden max-h-[400px] overflow-y-auto">
                                <table className="w-full text-sm text-left">
                                    <thead className="bg-slate-50 text-slate-600 font-medium sticky top-0">
                                        <tr>
                                            <th className="px-4 py-3">Type</th>
                                            <th className="px-4 py-3">Duration</th>
                                            <th className="px-4 py-3">Dates</th>
                                            <th className="px-4 py-3">Status</th>
                                        </tr>
                                    </thead>
                                    <tbody className="divide-y divide-slate-100">
                                        {leaveHistory.map((req) => (
                                            <tr key={req.id} className="hover:bg-slate-50">
                                                <td className="px-4 py-3 capitalize">{req.leave_type}</td>
                                                <td className="px-4 py-3 font-medium">{req.duration_days} Day(s)</td>
                                                <td className="px-4 py-3 text-slate-600">
                                                    {new Date(req.start_date).toLocaleDateString()} - {new Date(req.end_date).toLocaleDateString()}
                                                </td>
                                                <td className="px-4 py-3">
                                                    <span className={`inline-flex items-center px-2 py-1 rounded-full text-xs font-medium 
                                                        ${req.status === 'approved' ? 'bg-green-100 text-green-700' :
                                                            req.status === 'rejected' ? 'bg-red-100 text-red-700' :
                                                                req.status === 'pending' ? 'bg-amber-100 text-amber-700' :
                                                                    'bg-slate-100 text-slate-700'}`}>
                                                        {req.status.charAt(0).toUpperCase() + req.status.slice(1)}
                                                    </span>
                                                </td>
                                            </tr>
                                        ))}
                                    </tbody>
                                </table>
                            </div>
                        )}
                    </div>
                )}
            </div>
        </Modal>
    );
};

export default UserManagement;
