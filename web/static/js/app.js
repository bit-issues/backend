document.addEventListener('alpine:init', () => {
    Alpine.store('auth', {
        token: localStorage.getItem('token') || '',
        user: JSON.parse(localStorage.getItem('user') || 'null'),
        email: '',
        password: '',
        error: '',

        async login() {
            this.error = '';

            const response = await fetch('/api/v1/auth/login', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ email: this.email, password: this.password })
            });

            if (!response.ok) {
                this.error = 'Login failed';
                return false;
            }

            const data = await response.json();
            this.token = data.access_token || '';
            this.user = data.user || null;
            localStorage.setItem('token', this.token);
            localStorage.setItem('user', JSON.stringify(this.user));
            this.password = '';
            return true;
        },

        logout() {
            this.token = '';
            this.user = null;
            this.password = '';
            this.error = '';
            localStorage.removeItem('token');
            localStorage.removeItem('user');
        }
    });
});

async function apiFetch(path, options = {}) {
    const headers = options.headers || {};
    const token = Alpine.store('auth').token;

    if (token) {
        headers.Authorization = `Bearer ${token}`;
    }

    if (options.body && !(options.body instanceof FormData) && !headers['Content-Type']) {
        headers['Content-Type'] = 'application/json';
    }

    const response = await fetch(path, {
        ...options,
        headers
    });

    const text = await response.text();
    const data = text ? JSON.parse(text) : null;

    if (!response.ok) {
        throw new Error((data && data.message) || `Request failed (${response.status})`);
    }

    return data;
}

function taskDashboard() {
    return {
        view: 'tasks',
        adminSection: 'users',
        scope: 'my',
        tasks: [],
        total: 0,
        limit: 20,
        offset: 0,
        loading: false,
        error: '',
        sort: '-created_at',
        filters: {
            statuses: [],
            priorities: [],
            dueFrom: '',
            dueTo: '',
            project: ''
        },
        statusOptions: ['New', 'Open', 'In Progress', 'Resolved', 'Closed', 'Reopened'],
        priorityOptions: ['Trivial', 'Minor', 'Major', 'Critical', 'Blocker'],

        selectedTask: null,
        selectedStatus: '',
        newComment: '',
        pendingFile: null,
        uploadStatus: '',
        users: [],
        usersTotal: 0,
        usersOffset: 0,
        usersError: '',
        projectOptions: [],
        adminProjects: [],
        projectsTotal: 0,
        projectsOffset: 0,
        projectsError: '',
        projectFormOpen: false,
        editProjectID: '',
        projectForm: {
            name: '',
            repo_url: ''
        },

        formOpen: false,
        editTaskId: null,
        taskForm: {
            project_slug: '',
            title: '',
            description: '',
            priority: 'Minor',
            due_date: ''
        },

        init() {
            if (Alpine.store('auth').token) {
                this.fetchTasks();
                this.fetchProjects();
            }
        },

        async loginAndLoad() {
            const ok = await Alpine.store('auth').login();
            if (ok) {
                this.view = 'tasks';
                this.adminSection = 'users';
                this.fetchTasks();
                this.fetchProjects();
            }
        },

        async fetchTasks() {
            if (!Alpine.store('auth').token) {
                this.tasks = [];
                this.total = 0;
                this.error = 'Please login';
                return;
            }

            this.loading = true;
            this.error = '';

            const params = new URLSearchParams();
            params.set('limit', String(this.limit));
            params.set('offset', String(this.offset));
            params.set('sort', this.sort);

            if (this.filters.statuses.length > 0) {
                params.set('statuses', this.filters.statuses.join(','));
            }
            if (this.filters.priorities.length > 0) {
                params.set('priorities', this.filters.priorities.join(','));
            }
            if (this.filters.dueFrom) {
                params.set('due_from', this.filters.dueFrom);
            }
            if (this.filters.dueTo) {
                params.set('due_to', this.filters.dueTo);
            }
            if (this.filters.project) {
                params.set('project', this.filters.project);
            }

            const endpoint = this.scope === 'my' ? '/api/v1/tasks/me' : '/api/v1/tasks';

            try {
                const data = await apiFetch(`${endpoint}?${params.toString()}`);
                this.tasks = data.items || [];
                this.total = data.total || 0;
            } catch (e) {
                this.error = e.message;
                this.tasks = [];
                this.total = 0;
            } finally {
                this.loading = false;
            }
        },

        applyFilters() {
            this.offset = 0;
            this.fetchTasks();
        },

        nextPage() {
            if (this.offset + this.limit >= this.total) {
                return;
            }
            this.offset += this.limit;
            this.fetchTasks();
        },

        prevPage() {
            if (this.offset === 0) {
                return;
            }
            this.offset = Math.max(0, this.offset - this.limit);
            this.fetchTasks();
        },

        async openTask(id) {
            this.error = '';
            try {
                const data = await apiFetch(`/api/v1/tasks/${id}`);
                this.selectedTask = data;
                this.selectedStatus = data.status;
                this.newComment = '';
                this.uploadStatus = '';
            } catch (e) {
                this.error = e.message;
            }
        },

        openCreateForm() {
            this.editTaskId = null;
            this.taskForm = {
                project_slug: this.projectOptions.length > 0 ? this.projectOptions[0].id : '',
                title: '',
                description: '',
                priority: 'Minor',
                due_date: ''
            };
            this.formOpen = true;
        },

        openEditForm() {
            if (!this.selectedTask) {
                return;
            }
            this.editTaskId = this.selectedTask.id;
            this.taskForm = {
                project_slug: this.selectedTask.project_slug,
                title: this.selectedTask.title,
                description: this.selectedTask.description || '',
                priority: this.selectedTask.priority,
                due_date: this.selectedTask.due_date || ''
            };
            this.formOpen = true;
        },

        async submitTaskForm() {
            this.error = '';

            const body = {
                title: this.taskForm.title,
                description: this.taskForm.description,
                priority: this.taskForm.priority,
                due_date: this.taskForm.due_date || null
            };

            try {
                if (this.editTaskId) {
                    await apiFetch(`/api/v1/tasks/${this.editTaskId}`, {
                        method: 'PATCH',
                        body: JSON.stringify(body)
                    });
                    this.formOpen = false;
                    await this.openTask(this.editTaskId);
                } else {
                    await apiFetch('/api/v1/tasks', {
                        method: 'POST',
                        body: JSON.stringify({
                            ...body,
                            project_slug: this.taskForm.project_slug
                        })
                    });
                    this.formOpen = false;
                }

                await this.fetchTasks();
            } catch (e) {
                this.error = e.message;
            }
        },

        async updateStatus() {
            if (!this.selectedTask) {
                return;
            }

            try {
                await apiFetch(`/api/v1/tasks/${this.selectedTask.id}`, {
                    method: 'PATCH',
                    body: JSON.stringify({ status: this.selectedStatus })
                });
                await this.openTask(this.selectedTask.id);
                await this.fetchTasks();
            } catch (e) {
                this.error = e.message;
            }
        },

        async addComment() {
            if (!this.selectedTask || !this.newComment) {
                return;
            }

            try {
                await apiFetch(`/api/v1/tasks/${this.selectedTask.id}/comments`, {
                    method: 'POST',
                    body: JSON.stringify({ content: this.newComment })
                });
                await this.openTask(this.selectedTask.id);
            } catch (e) {
                this.error = e.message;
            }
        },

        selectFile(event) {
            const files = event.target.files || [];
            this.pendingFile = files.length > 0 ? files[0] : null;
            this.uploadStatus = '';
        },

        async uploadAttachment() {
            if (!this.selectedTask || !this.pendingFile) {
                return;
            }

            this.uploadStatus = 'Uploading...';

            try {
                const init = await apiFetch(`/api/v1/tasks/${this.selectedTask.id}/attachments`, {
                    method: 'POST',
                    body: JSON.stringify({
                        file_name: this.pendingFile.name,
                        size_bytes: this.pendingFile.size
                    })
                });

                await fetch(init.upload_url, {
                    method: 'PUT',
                    body: this.pendingFile
                });

                await apiFetch(`/api/v1/tasks/${this.selectedTask.id}/attachments/${init.id}/confirm`, {
                    method: 'PUT'
                });

                this.pendingFile = null;
                this.uploadStatus = 'Upload complete';
                await this.openTask(this.selectedTask.id);
            } catch (e) {
                this.uploadStatus = e.message;
            }
        },

        openAdmin() {
            this.view = 'admin';
            this.adminSection = 'users';
            this.usersOffset = 0;
            this.fetchUsers();
        },

        openTasks() {
            this.view = 'tasks';
        },

        async fetchUsers() {
            this.usersError = '';
            try {
                const data = await apiFetch(`/api/v1/users?limit=20&offset=${this.usersOffset}`);
                this.users = data.items || [];
                this.usersTotal = data.total || 0;
            } catch (e) {
                this.usersError = e.message;
                this.users = [];
                this.usersTotal = 0;
            }
        },

        async fetchProjects() {
            this.projectsError = '';
            try {
                const data = await apiFetch('/api/v1/projects?limit=100&offset=0');
                this.projectOptions = data.items || [];
            } catch (e) {
                this.projectsError = e.message;
                this.projectOptions = [];
            }
        },

        openProjectsAdmin() {
            this.adminSection = 'projects';
            this.projectsOffset = 0;
            this.fetchAdminProjects();
        },

        openUsersAdmin() {
            this.adminSection = 'users';
            this.usersOffset = 0;
            this.fetchUsers();
        },

        async fetchAdminProjects() {
            this.projectsError = '';
            try {
                const data = await apiFetch(`/api/v1/projects?limit=20&offset=${this.projectsOffset}`);
                this.adminProjects = data.items || [];
                this.projectsTotal = data.total || 0;
            } catch (e) {
                this.projectsError = e.message;
                this.adminProjects = [];
                this.projectsTotal = 0;
            }
        },

        openCreateProjectForm() {
            this.editProjectID = '';
            this.projectForm = { name: '', repo_url: '' };
            this.projectFormOpen = true;
        },

        openEditProjectForm(project) {
            this.editProjectID = project.id;
            this.projectForm = {
                name: project.name,
                repo_url: project.repo_url
            };
            this.projectFormOpen = true;
        },

        async submitProjectForm() {
            this.projectsError = '';
            try {
                if (this.editProjectID) {
                    await apiFetch(`/api/v1/projects/${this.editProjectID}`, {
                        method: 'PATCH',
                        body: JSON.stringify(this.projectForm)
                    });
                } else {
                    await apiFetch('/api/v1/projects', {
                        method: 'POST',
                        body: JSON.stringify(this.projectForm)
                    });
                }
                this.projectFormOpen = false;
                await this.fetchAdminProjects();
                await this.fetchProjects();
            } catch (e) {
                this.projectsError = e.message;
            }
        },

        async deleteProject(projectID) {
            if (!confirm('Delete project?')) {
                return;
            }
            this.projectsError = '';
            try {
                await apiFetch(`/api/v1/projects/${projectID}`, { method: 'DELETE' });
                await this.fetchAdminProjects();
                await this.fetchProjects();
            } catch (e) {
                this.projectsError = e.message;
            }
        },

        async updateUser(user) {
            this.usersError = '';
            try {
                await apiFetch(`/api/v1/users/${user.id}`, {
                    method: 'PATCH',
                    body: JSON.stringify({ status: user.status, role: user.role })
                });
            } catch (e) {
                this.usersError = e.message;
            }
        },

        nextUsersPage() {
            if (this.usersOffset + 20 >= this.usersTotal) {
                return;
            }
            this.usersOffset += 20;
            this.fetchUsers();
        },

        prevUsersPage() {
            if (this.usersOffset === 0) {
                return;
            }
            this.usersOffset = Math.max(0, this.usersOffset - 20);
            this.fetchUsers();
        },

        nextProjectsPage() {
            if (this.projectsOffset + 20 >= this.projectsTotal) {
                return;
            }
            this.projectsOffset += 20;
            this.fetchAdminProjects();
        },

        prevProjectsPage() {
            if (this.projectsOffset === 0) {
                return;
            }
            this.projectsOffset = Math.max(0, this.projectsOffset - 20);
            this.fetchAdminProjects();
        }
    };
}
