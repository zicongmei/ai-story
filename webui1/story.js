document.addEventListener('DOMContentLoaded', () => {
    const apiKeyInput = document.getElementById('apiKey');
    const modelSelect = document.getElementById('modelSelect');
    const systemInstructionTextarea = document.getElementById('systemInstruction');
    const nextParagraphPromptTextarea = document.getElementById('nextParagraphPrompt');
    const storyOutputTextarea = document.getElementById('storyOutput');
    const generateBtn = document.getElementById('generateBtn');
    const stopBtn = document.getElementById('stopBtn'); // Reference to the stop button
    const revertLastParagraphBtn = document.getElementById('revertLastParagraphBtn'); 
    const clearAllBtn = document.getElementById('clearAllBtn');
    const loadingIndicator = document.getElementById('loadingIndicator');
    const errorDisplay = document.getElementById('errorDisplay');
    const clearNextParagraphPromptBtn = document.getElementById('clearNextParagraphPromptBtn'); // New: Reference to the clear prompt button

    // Token display elements
    const currentRequestInputTokensDisplay = document.getElementById('currentRequestInputTokens');
    const currentRequestOutputTokensDisplay = document.getElementById('currentRequestOutputTokens');
    const accumulatedInputTokensDisplay = document.getElementById('accumulatedInputTokens'); 
    const accumulatedOutputTokensDisplay = document.getElementById('accumulatedOutputTokens'); 
    const accumulatedTokensDisplay = document.getElementById('accumulatedTokens'); 

    // Debug elements
    const debugToggleBtn = document.getElementById('debugToggleBtn');
    const debugPanel = document.getElementById('debugPanel');
    const debugLogsContainer = document.getElementById('debugLogs');
    const clearDebugLogsBtn = document.getElementById('clearDebugLogsBtn');

    // Global array to store request/response pairs for debugging
    const geminiLogs = [];

    // Global AbortController for stopping fetch requests
    let abortController = null;

    // Load accumulated tokens from localStorage, default to 0 if not found
    let totalAccumulatedInputTokens = parseInt(localStorage.getItem('geminiTotalAccumulatedInputTokens') || '0', 10);
    let totalAccumulatedOutputTokens = parseInt(localStorage.getItem('geminiTotalAccumulatedOutputTokens') || '0', 10);
    let totalAccumulatedTokens = totalAccumulatedInputTokens + totalAccumulatedOutputTokens;

    const GEMINI_API_BASE_URL = 'https://generativelanguage.googleapis.com/v1beta/models/';

    const defaultSystemInstruction = `You are a skilled story writer.
Continue the story one paragraph at a time, keeping the tone consistent.
Ensure the new paragraph naturally follows the existing text and incorporates the given prompt for the next part of the story.
Use the same language as input or previous paragraph.`;
    // Load saved settings from localStorage
    apiKeyInput.value = localStorage.getItem('geminiApiKey') || '';
    modelSelect.value = localStorage.getItem('geminiModel') || 'gemini-2.5-flash-lite';
    systemInstructionTextarea.value = localStorage.getItem('geminiSystemInstruction') || defaultSystemInstruction;
    nextParagraphPromptTextarea.value = localStorage.getItem('geminiNextParagraphPrompt') || '';
    storyOutputTextarea.value = localStorage.getItem('geminiStoryOutput') || ''; 
    
    // Display loaded accumulated tokens
    accumulatedInputTokensDisplay.textContent = totalAccumulatedInputTokens;
    accumulatedOutputTokensDisplay.textContent = totalAccumulatedOutputTokens;
    if (accumulatedTokensDisplay) { 
        accumulatedTokensDisplay.textContent = totalAccumulatedTokens;
    }

    // Initialize button state based on loaded story content.
    revertLastParagraphBtn.disabled = !storyOutputTextarea.value.trim();

    // Save settings to localStorage on change
    apiKeyInput.addEventListener('input', () => localStorage.setItem('geminiApiKey', apiKeyInput.value));
    modelSelect.addEventListener('change', () => localStorage.setItem('geminiModel', modelSelect.value));
    systemInstructionTextarea.addEventListener('input', () => localStorage.setItem('geminiSystemInstruction', systemInstructionTextarea.value));
    nextParagraphPromptTextarea.addEventListener('input', () => localStorage.setItem('geminiNextParagraphPrompt', nextParagraphPromptTextarea.value));
    
    // Save story on manual input and update button state
    storyOutputTextarea.addEventListener('input', () => {
        localStorage.setItem('geminiStoryOutput', storyOutputTextarea.value);
        revertLastParagraphBtn.disabled = !storyOutputTextarea.value.trim(); // Update button state on manual edit
    });

    // New: Event listener for clearing nextParagraphPrompt
    clearNextParagraphPromptBtn.addEventListener('click', () => {
        nextParagraphPromptTextarea.value = '';
        localStorage.removeItem('geminiNextParagraphPrompt'); // Also clear from local storage
    });

    generateBtn.addEventListener('click', generateParagraph);
    revertLastParagraphBtn.addEventListener('click', removeLastParagraph); // This function will now remove the last paragraph
    clearAllBtn.addEventListener('click', clearAllContents);

    // Stop button event listener
    stopBtn.addEventListener('click', () => {
        if (abortController) {
            abortController.abort(); // Signal to abort the fetch request
            showError('Generation stopped by user.'); // Inform the user
            // Immediately update UI to reflect stopping
            generateBtn.disabled = false;
            generateBtn.classList.remove('hidden');
            stopBtn.classList.add('hidden');
            loadingIndicator.classList.add('hidden');
            abortController = null; // Clear the controller
            revertLastParagraphBtn.disabled = !storyOutputTextarea.value.trim(); // Re-evaluate based on current content
        }
    });

    // Debug panel event listeners
    debugToggleBtn.addEventListener('click', () => {
        debugPanel.classList.toggle('hidden');
        if (!debugPanel.classList.contains('hidden')) {
            debugLogsContainer.scrollTop = debugLogsContainer.scrollHeight; // Scroll to bottom on open
        }
    });

    clearDebugLogsBtn.addEventListener('click', () => {
        if (confirm('Are you sure you want to clear all debug logs?')) {
            geminiLogs.length = 0; // Clear the array
            debugLogsContainer.innerHTML = ''; // Clear the display
        }
    });

    function clearAllContents() {
        if (!confirm('Are you sure you want to clear all contents and settings (except API key)? This cannot be undone.')) {
            return;
        }

        modelSelect.value = 'gemini-2.5-flash-lite'; 
        systemInstructionTextarea.value = defaultSystemInstruction; 
        nextParagraphPromptTextarea.value = ''; // Clear next paragraph prompt as well
        storyOutputTextarea.value = '';

        revertLastParagraphBtn.disabled = true;

        // Clear localStorage (except API key)
        localStorage.removeItem('geminiModel');
        localStorage.removeItem('geminiSystemInstruction');
        localStorage.removeItem('geminiNextParagraphPrompt');
        localStorage.removeItem('geminiStoryOutput'); 
        localStorage.removeItem('geminiTotalAccumulatedInputTokens'); 
        localStorage.removeItem('geminiTotalAccumulatedOutputTokens'); 

        // Clear token displays
        totalAccumulatedInputTokens = 0; 
        totalAccumulatedOutputTokens = 0; 
        totalAccumulatedTokens = 0; 
        currentRequestInputTokensDisplay.textContent = '0';
        currentRequestOutputTokensDisplay.textContent = '0';
        if (accumulatedInputTokensDisplay) { accumulatedInputTokensDisplay.textContent = '0'; }
        if (accumulatedOutputTokensDisplay) { accumulatedInputTokensDisplay.textContent = '0'; }
        if (accumulatedTokensDisplay) { 
            accumulatedTokensDisplay.textContent = '0';
        }

        showError(''); 
    }

    // This function is now responsible for removing the last paragraph directly from the textbox.
    function removeLastParagraph() {
        let currentStory = storyOutputTextarea.value.trim();
        if (!currentStory) {
            revertLastParagraphBtn.disabled = true;
            return;
        }

        // Split by two or more newlines to identify distinct paragraphs.
        // Trim each part and filter out any empty strings resulting from the split.
        let paragraphs = currentStory.split(/\n\n/).map(p => p.trim()).filter(p => p !== '');

        if (paragraphs.length > 0) {
            paragraphs.pop(); // Remove the last actual paragraph
            storyOutputTextarea.value = paragraphs.join('\n\n');
            localStorage.setItem('geminiStoryOutput', storyOutputTextarea.value);
            
            // Re-evaluate button state based on the new content
            revertLastParagraphBtn.disabled = !storyOutputTextarea.value.trim();
            storyOutputTextarea.scrollTop = storyOutputTextarea.scrollHeight;
        } else {
            // If there were no discernible paragraphs left after splitting/filtering
            storyOutputTextarea.value = '';
            localStorage.setItem('geminiStoryOutput', '');
            revertLastParagraphBtn.disabled = true;
        }
    }

    async function generateParagraph() {
        const apiKey = apiKeyInput.value.trim();
        const selectedModel = modelSelect.value;
        const systemInstruction = systemInstructionTextarea.value.trim();
        const currentStory = storyOutputTextarea.value.trim();
        const nextParagraphPrompt = nextParagraphPromptTextarea.value.trim();

        if (!apiKey) {
            showError('Please enter your Gemini API Key.');
            return;
        }
        
        if (abortController) { // Prevent starting a new generation if one is already active
            showError('Another generation is already in progress. Please wait or stop it first.');
            return;
        }

        abortController = new AbortController(); // Initialize AbortController for this request
        const signal = abortController.signal;

        generateBtn.disabled = true;
        generateBtn.classList.add('hidden'); // Hide generate button
        stopBtn.classList.remove('hidden'); // Show stop button
        revertLastParagraphBtn.disabled = true; // Disable during generation
        loadingIndicator.classList.remove('hidden');
        showError(''); 

        currentRequestInputTokensDisplay.textContent = 'Calculating...'; 
        currentRequestOutputTokensDisplay.textContent = 'Calculating...'; 
        
        let userPrompt = '';
        if (currentStory === '') {
            userPrompt = `Start a new story. The first paragraph should be about: ${nextParagraphPrompt}`;
        } else {
            userPrompt = `Here is the story so far:\n\n${currentStory}\n\nWhat should happen next is: ${nextParagraphPrompt}\n\nContinue the story with ONE new paragraph, making sure it logically follows the previous text and incorporates the "what should happen next" prompt.`;
        }
        
        const requestBody = {
            contents: [{
                role: 'user',
                parts: [{ text: userPrompt }]
            }],
            generationConfig: {
                temperature: 0.9,
                topP: 1,
                topK: 1,
                // maxOutputTokens: 500, 
            },
            safetySettings:[{
                category: 'HARM_CATEGORY_HATE_SPEECH',
                threshold: 'OFF',
            }, {
                category: 'HARM_CATEGORY_SEXUALLY_EXPLICIT',
                threshold: 'OFF',
            }, {
                category: 'HARM_CATEGORY_DANGEROUS_CONTENT',
                threshold: 'OFF',
            }, {
                category: 'HARM_CATEGORY_HARASSMENT',
                threshold: 'OFF',
            }, {
                category: 'HARM_CATEGORY_CIVIC_INTEGRITY',
                threshold: 'OFF',
            }],
        };

        if (systemInstruction) {
            requestBody.systemInstruction = {
                parts: [{ text: systemInstruction }]
            };
        }

        const requestBodyString = JSON.stringify(requestBody); // Store stringified version for debug log

        try {
            const response = await fetch(`${GEMINI_API_BASE_URL}${selectedModel}:generateContent?key=${apiKey}`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: requestBodyString,
                signal: signal, // Pass the AbortController's signal here
            });

            if (!response.ok) {
                const errorData = await response.json();
                appendDebugLog(requestBodyString, errorData); // Log error response
                throw new Error(errorData.error ? errorData.error.message : response.statusText);
            }

            const data = await response.json();
            appendDebugLog(requestBodyString, data); // Log successful response

            const generatedText = data.candidates[0]?.content?.parts[0]?.text;

            const promptTokens = data.usageMetadata?.promptTokenCount || 0;
            const candidateTokens = data.usageMetadata?.candidatesTokenCount || 0; 

            currentRequestInputTokensDisplay.textContent = promptTokens;
            currentRequestOutputTokensDisplay.textContent = candidateTokens;
            
            totalAccumulatedInputTokens += promptTokens;
            totalAccumulatedOutputTokens += candidateTokens;
            totalAccumulatedTokens = totalAccumulatedInputTokens + totalAccumulatedOutputTokens; 

            accumulatedInputTokensDisplay.textContent = totalAccumulatedInputTokens;
            accumulatedOutputTokensDisplay.textContent = totalAccumulatedOutputTokens;
            if (accumulatedTokensDisplay) { 
                accumulatedTokensDisplay.textContent = totalAccumulatedTokens;
            }

            localStorage.setItem('geminiTotalAccumulatedInputTokens', totalAccumulatedInputTokens.toString());
            localStorage.setItem('geminiTotalAccumulatedOutputTokens', totalAccumulatedOutputTokens.toString());

            if (generatedText) {
                if (storyOutputTextarea.value.trim() === '') {
                    storyOutputTextarea.value = generatedText.trim();
                } else {
                    storyOutputTextarea.value += '\n\n' + generatedText.trim();
                }
                localStorage.setItem('geminiStoryOutput', storyOutputTextarea.value); 
                // nextParagraphPromptTextarea.value = ''; // Removed as per request: do not clear nextParagraphPrompt
                storyOutputTextarea.scrollTop = storyOutputTextarea.scrollHeight;
            } else {
                showError('No content generated. The model might have been blocked due to safety concerns or returned an empty response.');
                currentRequestInputTokensDisplay.textContent = '0'; 
                currentRequestOutputTokensDisplay.textContent = '0'; 
            }

        } catch (error) {
            console.error('Error calling Gemini API:', error);
            if (error.name === 'AbortError') {
                showError('Generation stopped by user.');
            } else {
                showError(`Failed to generate paragraph: ${error.message}`);
                if (!error.isLoggedAsResponse) {
                    appendDebugLog(requestBodyString, error);
                }
            }
            currentRequestInputTokensDisplay.textContent = '0'; 
            currentRequestOutputTokensDisplay.textContent = '0'; 
        } finally {
            generateBtn.disabled = false;
            generateBtn.classList.remove('hidden'); // Show generate button
            stopBtn.classList.add('hidden'); // Hide stop button
            loadingIndicator.classList.add('hidden');
            abortController = null; // Clear the controller
            revertLastParagraphBtn.disabled = !storyOutputTextarea.value.trim(); // Re-evaluate based on current content
        }
    }

    // Function to append a new log entry
    function appendDebugLog(requestBodyString, responseOrError) {
        geminiLogs.push({
            timestamp: new Date().toLocaleString(),
            request: JSON.parse(requestBodyString), // Parse back to object for display
            response: responseOrError
        });
        renderDebugLogs();
    }

    // Function to render all logs in the debug panel
    function renderDebugLogs() {
        debugLogsContainer.innerHTML = ''; // Clear previous logs
        geminiLogs.forEach((log, index) => {
            const logEntryDiv = document.createElement('div');
            logEntryDiv.classList.add('debug-log-entry');

            // Request summary
            const requestSummary = document.createElement('details');
            requestSummary.innerHTML = `<summary><strong>Request #${index + 1}</strong> (${log.timestamp})</summary>`;
            const requestPre = document.createElement('pre');
            requestPre.textContent = JSON.stringify(log.request, null, 2);
            requestSummary.appendChild(requestPre);
            logEntryDiv.appendChild(requestSummary);

            // Response summary
            const responseSummary = document.createElement('details');
            const responseData = log.response instanceof Error ? 
                                 { error: log.response.message, stack: log.response.stack } : 
                                 log.response;
            responseSummary.innerHTML = `<summary><strong>Response #${index + 1}</strong></summary>`;
            const responsePre = document.createElement('pre');
            responsePre.textContent = JSON.stringify(responseData, null, 2);
            responseSummary.appendChild(responsePre);
            logEntryDiv.appendChild(responseSummary);

            debugLogsContainer.appendChild(logEntryDiv);
        });
        debugLogsContainer.scrollTop = debugLogsContainer.scrollHeight; // Scroll to bottom
    }

    function showError(message) {
        if (message) {
            errorDisplay.textContent = message;
            errorDisplay.classList.remove('hidden');
        } else {
            errorDisplay.textContent = '';
            errorDisplay.classList.add('hidden');
        }
    }
});