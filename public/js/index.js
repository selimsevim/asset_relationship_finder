document.addEventListener('DOMContentLoaded', function () {
            const assetTypeSelect = document.getElementById('assetTypeSelect');
            const dataExtensionForm = document.getElementById('dataExtensionForm');
            const activityForm = document.getElementById('activityForm');
            const emailForm = document.getElementById('emailForm'); 
            const cloudPageForm = document.getElementById('cloudPageForm');
            const deSubmitBtn = document.getElementById('deSubmitBtn');
            const emailSubmitBtn = document.getElementById('emailSubmitBtn');
            const cloudPageSubmitBtn = document.getElementById('cloudPageSubmitBtn');
            const activitySubmitBtn = document.getElementById('activitySubmitBtn');
            const errorMessage = document.getElementById('errorMessage');
            const resultsPlaceholder = document.getElementById('resultsPlaceholder');
            const resultSection = document.getElementById('resultsContent');
            const goTopBtn = document.getElementById('goTopBtn');
            const selectedAsset = document.getElementById('selectedAsset');

            const formMap = {
                dataextensions: dataExtensionForm.classList,
                queries: activityForm.classList,
                scripts: activityForm.classList,
                importactivities: activityForm.classList,
                filteractivities: activityForm.classList,
                cloudpages: cloudPageForm.classList,
                emails: emailForm.classList
            };
            
            // Show the "Go to Top" button based on scrolling position
            window.addEventListener('scroll', function () {
                const scrollPosition = window.scrollY || document.documentElement.scrollTop;
                const windowHeight = document.documentElement.scrollHeight - window.innerHeight;

                if (scrollPosition > windowHeight * 0.2) {
                    goTopBtn.style.display = 'block';
                } else {
                    goTopBtn.style.display = 'none';
                }
            });

            // Scroll to top when "Go to Top" button is clicked
            goTopBtn.addEventListener('click', function () {
                window.scrollTo({ top: 0, behavior: 'smooth' });
            });

            // Show/hide related asset form based on asset type selection
            assetTypeSelect.addEventListener('change', function () {
                const selectedText = this.value;  // Capture the original value for display
                selectedAsset.textContent = selectedText.replace(/([A-Z])/g, ' $1').trim();

                // Normalize selected value for comparisons (lowercase and no spaces)
                const selectedValue = selectedText.toLowerCase().replace(/\s+/g, '');

                // Hide all forms first
                for (let key in formMap) {
                    formMap[key].add('d-none');
                }

                // Show the form associated with the selected asset type if it exists
                if (formMap[selectedValue]) {
                    formMap[selectedValue].remove('d-none');
                }

                resetResults(); // Reset results
                resultsPlaceholder.classList.remove('hidden');
                resultsPlaceholder.innerHTML = 'Information will be shown here.';
                errorMessage.classList.add('d-none');
            });

            // Submit handler for activities (queries, scripts, etc.)
            activitySubmitBtn.addEventListener('click', async function () {
                await handleFormSubmit('activity');
            });

            // Submit handler for data extension
            deSubmitBtn.addEventListener('click', async function () {
                await handleFormSubmit('dataExtension');
            });

            // Submit handler for Cloudpages
            cloudPageSubmitBtn.addEventListener('click', async function () {
                await handleFormSubmit('cloudPage');
            });

            // Submit handler for Emails
            emailSubmitBtn.addEventListener('click', async function () {
                await handleFormSubmit('email');  // New handler for email submissions
            });

            // Generalized function for handling form submission
            async function handleFormSubmit(type) {
                resetResults();
                errorMessage.classList.add('d-none');
                
                const submitBtn = type === 'dataExtension' ? deSubmitBtn : 
                                  type === 'cloudPage' ? cloudPageSubmitBtn : 
                                  type === 'email' ? emailSubmitBtn : activitySubmitBtn;


                submitBtn.disabled = true;

                let inputKeyValue;
                if (type === 'dataExtension') {
                    inputKeyValue = document.getElementById('deNameKey').value.trim();  
                } else if (type === 'cloudPage') {
                    inputKeyValue = document.getElementById('cloudPageID').value.trim(); 
                } else if (type === 'email') {
                    inputKeyValue = document.getElementById('emailInput').value.trim();  
                } else {
                    inputKeyValue = document.getElementById('activityNameKey').value.trim(); 
                }

                if (!inputKeyValue) {
                    errorMessage.innerHTML = `Please enter a ${type === 'dataExtension' ? 'Data Extension' : 
                                              type === 'email' ? 'Email' : 
                                              type === 'cloudPage' ? 'CloudPage' : 'Activity'} Name or ID/Key.`;

                    errorMessage.classList.remove('d-none');
                    submitBtn.disabled = false;
                    resultsPlaceholder.classList.add('hidden');
                    return;
                }

                const requestData = buildRequestData(type, inputKeyValue);

                // Only check for selected checkboxes if the type is 'dataExtension'
                let selectedCheckboxes = {};
                let hasSelection = true; // Default to true for non-dataExtension types

                if (type === 'dataExtension' || type === 'cloudPage' || type === 'email') {

                    hasSelection = false;
                    document.querySelectorAll(`#${type}Form input[type="checkbox"]`).forEach(checkbox => {
                        selectedCheckboxes[checkbox.value] = checkbox.checked;
                        if (checkbox.checked) {
                            hasSelection = true;
                        }
                    });

                    if (!hasSelection) {
                        errorMessage.innerHTML = 'Please select at least one option.';
                        errorMessage.classList.remove('d-none');
                        submitBtn.disabled = false;
                        resultsPlaceholder.classList.add('hidden');
                        return;
                    }
                }

                try {
                        const response = await fetch(`/${type === 'dataExtension' ? 'data-extension-detail' : 
                                                         type === 'cloudPage' ? 'cloud-page-detail' : 
                                                         type === 'email' ? 'email-detail' : 'automation-activity-detail'}`, {

                        method: 'POST',
                        headers: { 'Content-Type': 'application/json' },
                        body: JSON.stringify(requestData)
                    });

                    if (!response.ok) {
                        const errorText = await response.text();
                        handleError(errorText);
                        return;
                    }

                    const result = await response.json();
                    processResults(result, requestData, selectedCheckboxes, type); 

                } catch (error) {
                    handleError(`An error occurred while retrieving ${type === 'dataExtension' ? 'Data Extension' : 
                                                                      type === 'cloudPage' ? 'CloudPage' : 
                                                                      type === 'email' ? 'Email' : 'Activity'} details. Please try again.`);

                } finally {
                    submitBtn.disabled = false;
                }
            }

            // Function to build request data based on form inputs
            function buildRequestData(type, inputKeyValue) {
                let requestData = {};

                if (type === 'dataExtension') {
                    const deKeyType = document.getElementById('deKeySelect').value;
                    requestData = {
                        name: deKeyType === "Name" ? inputKeyValue : "",
                        customerKey: deKeyType === "CustomerKey" ? inputKeyValue : "",
                    };
                    const checkboxes = document.querySelectorAll('#dataExtensionForm input[type="checkbox"]');
                    const userSelection = {};
                    checkboxes.forEach(checkbox => {
                        userSelection[checkbox.value] = checkbox.checked;
                    });
                    requestData.userselection = userSelection;
                } else if (type === 'cloudPage') {
                    requestData = {
                        cloudPageID: inputKeyValue
                    };
                    const checkboxes = document.querySelectorAll('#cloudPageForm input[type="checkbox"]');
                    const userSelection = {};
                    checkboxes.forEach(checkbox => {
                        userSelection[checkbox.value] = checkbox.checked;
                    });
                    requestData.userselection = userSelection;
                } else if (type === 'email') {
                    const emailKeyType = document.getElementById('emailSelect').value;  // Email ID or Name
                    requestData = {
                        ID: emailKeyType === "ID" ? inputKeyValue : "",
                        Name: emailKeyType === "Name" ? inputKeyValue : ""
                    };
                    const checkboxes = document.querySelectorAll('#emailForm input[type="checkbox"]');
                    const userSelection = {};
                    checkboxes.forEach(checkbox => {
                        userSelection[checkbox.value] = checkbox.checked;
                    });
                    requestData.userselection = userSelection;
                } else {
                    const activityName = document.getElementById('activityNameKey').value;
                    requestData = {
                        name: activityName,
                        activityType: assetTypeSelect.value
                    };
                }

                console.log(JSON.stringify(requestData));
                return requestData;
            }

             // Function to process and display results based on selected checkboxes
            function processResults(result, requestData, selectedCheckboxes, type) {
                resultsPlaceholder.classList.add('hidden');
                resultSection.classList.remove('hidden');

                let resultHtml = '';

                console.log('Result object:', result);

                // Mapped arrays for different types of assets
                const dataExtensionKeyMap = [
                    { optionname: 'dePath', notfoundmsg: 'No path found for this Data Extension.', title: 'Path' },
                    { optionname: 'queriesTargeting', notfoundmsg: 'No queries found targeting this Data Extension.', title: 'Queries targeting this Data Extension' },
                    { optionname: 'queriesIncluding', notfoundmsg: 'No queries found including this Data Extension.', title: 'Queries including this Data Extension' },
                    { optionname: 'importsTargeting', notfoundmsg: 'No import activities found targeting this Data Extension.', title: 'Import activities targeting this Data Extension' },
                    { optionname: 'filtersTargeting', notfoundmsg: 'No filters found targeting this Data Extension.', title: 'Filters targeting this Data Extension' },
                    { optionname: 'contentEmailsIncluding', notfoundmsg: 'No Content Builder emails found using this Data Extension.', title: 'Content Builder emails using this Data Extension' },
                    { optionname: 'initiatedEmailsTargeting', notfoundmsg: 'No initiated emails found using this Data Extension.', title: 'Initiated emails using this Data Extension' },
                    { optionname: 'journeysUsingDE', notfoundmsg: 'No journeys found using this Data Extension.', title: 'Journeys using this Data Extension' },
                    { optionname: 'scriptsIncluding', notfoundmsg: 'No scripts found including this Data Extension.', title: 'Scripts including this Data Extension' },
                    { optionname: 'pagesIncluding', notfoundmsg: 'No CloudPages found including this Data Extension.', title: 'CloudPages including this Data Extension' }
                ];

                const cloudPageKeyMap = [
                    { optionname: 'emailsUsingCloudPage', notfoundmsg: 'No emails found using this CloudPage.', title: 'Emails using this CloudPage' },
                    { optionname: 'cloudPagesUsingCloudPage', notfoundmsg: 'No CloudPages found using this CloudPage.', title: 'CloudPages using this CloudPage' }
                ];

                const emailKeyMap = [
                    { optionname: 'journeysUsingEmail', notfoundmsg: 'No journeys found using this Email.', title: 'Journeys using this Email' },
                    { optionname: 'initiatedEmailsUsing', notfoundmsg: 'No User-Initiated Emails found using this Email.', title: 'User-Initiated Emails using this Email' },
                    { optionname: 'triggeredSends', notfoundmsg: 'No Triggered Sends found using this Email.', title: 'Triggered Sends using this Email' }
                ];

                // Check if the type is dataExtension with customerKey or email with ID, and display the Name
                if ((type === 'dataExtension' && requestData.customerKey && result.name) || 
                    (type === 'email' && requestData.ID && result.name)) {
                    resultHtml += `<div class="result-container"><h6 class="fw-bold">Name</h6><p>${result.name || result.name}</p></div>`;
                }

                // Use the appropriate keyMap based on the asset type
                let keyMap = [];
                if (type === 'dataExtension') {
                    keyMap = dataExtensionKeyMap;
                } else if (type === 'cloudPage') {
                    keyMap = cloudPageKeyMap;
                } else if (type === 'email') {
                    keyMap = emailKeyMap;  
                }

                // Iterate over the keyMap and process results based on selected checkboxes
                keyMap.forEach(item => {
                    const { optionname, notfoundmsg, title } = item;

                    if (selectedCheckboxes[optionname]) {
                        const data = result[optionname];

                        // Always show the title
                        resultHtml += `<div class="result-container">`;
                        resultHtml += `<h6 class="fw-bold">${title}</h6>`;

                        // For path (which is a string)
                        if (optionname === 'dePath') {
                            if (data) {
                                resultHtml += `<p>${data}</p>`;
                            } else {
                                resultHtml += `<p class="text-muted">${notfoundmsg}</p>`;
                            }
                        } 
                        // For non-empty array data
                        else if (data && Array.isArray(data) && data.length > 0) {
                            resultHtml += `<ul class="list-group">`;

                            // Show the first 5 items
                            data.slice(0, 5).forEach(item => {
                                resultHtml += `<li class="list-group-item">${item.Name || item}</li>`;
                            });

                            // Add remaining items with a 'hidden-item' class to hide them initially
                            data.slice(5).forEach(item => {
                                resultHtml += `<li class="list-group-item hidden-item">${item.Name || item}</li>`;
                            });

                            // Log the hidden items to verify
                            console.log(`Added hidden-item class to items for ${optionname}`);

                            // Add the View more/less button if there are more than 5 items
                            if (data.length > 5) {
                                resultHtml += `<button class="btn btn-link btn-sm toggle-btn" data-key="${optionname}">View more</button>`;
                            }

                            resultHtml += `</ul>`;
                        } else {
                            resultHtml += `<p class="text-muted">${notfoundmsg}</p>`;
                        }

                        resultHtml += `</div>`; // Close the container
                    }
                });

                // Handle the automation type separately
                if (type === 'activity') {
                    const automations = result.automations || [];
                    resultHtml += `<div class="result-container">`;
                    resultHtml += `<h6 class="fw-bold">Automations</h6>`;

                    if (automations.length > 0) {
                        resultHtml += `<ul class="list-group">`;
                        automations.forEach(automation => {
                            resultHtml += `<li class="list-group-item">${automation.Name}</li>`;
                        });
                        resultHtml += `</ul>`;
                    } else {
                        resultHtml += `<p class="text-muted">No automations found.</p>`;
                    }

                    resultHtml += `</div>`;
                }

                resultSection.innerHTML = resultHtml;

                // Attach click event for "View more/less" buttons only for dataExtension and cloudPage
                if (type === 'dataExtension' || type === 'cloudPage' || type === 'email') {
                    attachViewMoreListeners();
                }
            }

            // Function to attach listeners for "View more/less" buttons
            function attachViewMoreListeners() {
                document.querySelectorAll('.toggle-btn').forEach(button => {
                    button.addEventListener('click', function () {
                        // Find all list items, both visible and hidden
                        const allItems = button.closest('.result-container').querySelectorAll('.list-group-item');
                        
                        // Separate the first five (always visible) items and the remaining ones
                        const initialItems = Array.from(allItems).slice(0, 5);
                        const extraItems = Array.from(allItems).slice(5);

                        // Determine whether to show or hide the extra items
                        const isHidden = extraItems[0].classList.contains('hidden-item');

                        // Toggle visibility based on current state
                        extraItems.forEach(item => {
                            if (isHidden) {
                                item.classList.remove('hidden-item'); // Show items
                            } else {
                                item.classList.add('hidden-item'); // Hide items
                            }
                        });

                        // Change the button text based on the state
                        button.textContent = isHidden ? 'View less' : 'View more';
                    });
                });
            }


            // Function to handle and display errors
            function handleError(message) {
                errorMessage.innerHTML = message;
                errorMessage.classList.remove('d-none');
                resultsPlaceholder.classList.add('hidden');
            }

            // Function to reset the result UI
            function resetResults() {
                resultSection.classList.add('hidden');
                resultSection.innerHTML = '';
                resultsPlaceholder.classList.remove('hidden');
                resultsPlaceholder.innerHTML = 'Results are loading...';
            }
        });